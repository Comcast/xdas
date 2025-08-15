// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"log/slog"

	_ "go.uber.org/automaxprocs"

	"bytes"
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"xdas/internal/conversion"
	"xdas/internal/keyspaces"
	"xdas/internal/logger"
	"xdas/internal/rediscrypto"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v7"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// AppName overide the following through ldflags="-X <package>.<varName>=<value>"
	AppName    = "xdas"
	AppVersion = "unknown"
	BuildTime  = "unkown"
)

const (
	xdasAPIPath       = "/v2"
	defaultGlobalTTL  = time.Hour * 168
	defaultDMTTL      = time.Hour * 24 * 365
	defaultAccelDMTTL = time.Hour * 24 * 7

	GSKeyspace = "gs"
)

// A Server holds all the servers and configurations
type Server struct {
	config  *Configuration
	router  *chi.Mux
	web     *http.Server
	hClient *http.Client
	redis   redis.UniversalClient
	metrics *appMetrics
	bufPool sync.Pool
	log     *logger.Logger
}

func main() {
	logger := logger.NewLogger()
	slog.SetDefault(logger.Logger)

	config := getConfig(logger)

	if config.Verbose {
		logger.SetLevel(slog.LevelDebug)
	}

	logger.Info("Server is starting...")

	_, err := rediscrypto.Init("AesGCM", config.Redis.EncryptionKey)
	if err != nil {
		logger.Fatal(err)
	}

	redisClient := redis.NewUniversalClient(config.Redis.ClientConfig)
	s := &Server{
		config:  config,
		router:  chi.NewRouter(),
		hClient: config.HClient.Client,
		redis:   redisClient,
		metrics: newMetrics(),
		bufPool: sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
		log:     logger,
	}

	s.newWebServer()
	s.addRoutes()

	s.newConvert()
	s.newFindX()

	val := s.redis.ClusterInfo().Val()
	logger.Info("Redis cluster info: " + val)
	val = s.redis.ClusterNodes().Val()
	logger.Info("Redis cluster nodes: " + val)

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go s.shutdown(quit, done)

	logger.Info("Server is ready to handle requests", "addr", s.web.Addr)

	if tls := s.config.Web.TLS; tls.CertFile != "" && tls.KeyFile != "" {
		if err := s.web.ListenAndServeTLS(tls.CertFile, tls.KeyFile); err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on %s: %v\n", s.web.Addr, err)
		}
	} else {
		if err := s.web.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on %s: %v\n", s.web.Addr, err)
		}
	}

	<-done
	logger.Info("Shutdown complete")
}

func (s *Server) newWebServer() {
	// s.config.Web.Handler = h2c.NewHandler(s.router, &http2.Server{})
	s.config.Web.Server.Handler = s.router
	s.web = s.config.Web.Server
}

func (s *Server) newFindX() {
	s.log.Info("FindX is staring...")
	for keyspace, ksConf := range s.config.Keyspaces {
		if ksConf.FindX.Enabled {
			ksConf.FindX.Keyspace = keyspace
			ksConf.FindX.Metrics.Keyspace = keyspace
			ksConf.FindX.Metrics.PromNamespace = AppName
			ksConf.FindX.HTTPClient = s.hClient
			err := ksConf.FindX.Start()
			if err != nil {
				s.log.Error("Error starting FindX for", "keyspace", keyspace, "err", err)
			}
		}
	}
}

func (s *Server) newConvert() {
	keyspaces := make([]string, 0, len(s.config.Keyspaces))
	for k := range s.config.Keyspaces {
		keyspaces = append(keyspaces, k)
	}
	conversion.Init(prometheus.DefaultRegisterer, AppName, keyspaces)
}

func (s *Server) shutdown(quit chan os.Signal, done chan bool) {
	<-quit
	s.log.Info("Web server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.web.SetKeepAlivesEnabled(false)
	if err := s.web.Shutdown(ctx); err != nil {
		s.log.Info("Could not gracefully shutdown the server:", "err", err)
	}
	s.log.Info("Redis client is shutting down...")
	if err := s.redis.Close(); err != nil {
		s.log.Info("Failed to shut down Redis client cleanly", "err", err)
	}
	s.log.Info("FindX is shutting down...")
	for _, ksConf := range s.config.Keyspaces {
		ksConf.FindX.Close()
	}
	close(done)
}

// addURLParamKeyspace adds Chi keyspace parameter. Used for Prometheus metrics
// and keyspaces that is hardcoded
func addURLParamKeyspace(keyspace string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chi.RouteContext(r.Context()).URLParams.Add("keyspace", keyspace)
			next.ServeHTTP(w, r)
		})
	}
}

// validateKeyspace is a middleware that checks if a keyspace is validate
// it serves 2 purposes, both to validate and protect metrics from flood of invalid keyspaces
func (s *Server) validateKeyspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyspace := chi.URLParam(r, "keyspace")
		if _, ok := s.config.Keyspaces[keyspace]; !ok {
			s.log.Info("Invalid keyspace", "keyspace", keyspace)
			http.Error(w, "Invalid keyspace", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// validateAtomicKeyspace is a middleware that checks if a keyspace is validate and it's atomic
// it serves 2 purposes, both to validate and protect metrics from flood of invalid keyspaces
func (s *Server) validateAtomicKeyspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyspace := chi.URLParam(r, "keyspace")
		if c, ok := s.config.Keyspaces[keyspace]; !ok || c.Kind != keyspaces.KSAtomic {
			s.log.Info("Invalid keyspace or not atomic inc", "keyspace", keyspace)
			http.Error(w, "Invalid keyspace", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
