// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"xdas/internal/logger/weblog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MaxSize is the max message size we will accept
const MaxSize = 1000000

func (s *Server) addRoutes() {
	if s.config.Verbose {
		s.router.Use(weblog.WebLogChiMiddleware(s.log))
		// s.router.Use(s.webLogging)
	}
	s.router.Use(middleware.RequestSize(MaxSize))

	s.router.Route(xdasAPIPath, func(r chi.Router) {
		r.Route("/multi", func(r chi.Router) {
			r.Use(addURLParamKeyspace("multi"))
			if !s.config.NoMetrics {
				r.Use(s.metrics.appMetrics)
			}
			r.Get("/{id}", s.handleFuncXdasMultiGet)
			// r.Put("/{id}", s.handleFuncXdasMultiPut)
			// r.Post("/{id}", s.handleFuncXdasMultiPut)
		})
		r.Route("/{keyspace}", func(r chi.Router) {
			r.Use(s.validateKeyspace)
			if !s.config.NoMetrics {
				r.Use(s.metrics.appMetrics)
			}
			r.Get("/{id}", s.handleFuncXdasGet)
			r.Put("/{id}", s.handleFuncXdasPut)
			r.Post("/{id}", s.handleFuncXdasPut)
			r.Delete("/{id}", s.handleFuncXdasDel)
		})

		r.Route("/inc/{keyspace}", func(r chi.Router) {
			r.Use(s.validateAtomicKeyspace)
			if !s.config.NoMetrics {
				r.Use(s.metrics.appMetrics)
			}
			// r.Get("/{id}", s.handleFuncXdasGet)
			r.Put("/{id}", s.handleFuncXdasAtomicInc)
			r.Post("/{id}", s.handleFuncXdasAtomicInc)
			// r.Delete("/{id}", s.handleFuncXdasDel)
		})
	})

	s.router.Get("/metrics", promhttp.Handler().ServeHTTP)
	// s.router.Get("/config", s.handleConfig)
	s.router.Get("/version", s.handleVersion)
	s.router.Get("/healthz", s.handleHealthz)
}
