// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package config

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"time"
)

var ErrNoWebAddr = errors.New("missing addr")

type WebConfig struct {
	Server *http.Server
	TLS    TLS
}

var webAddr = flag.String("addr", os.Getenv("WEB_ADDR"), "The address to bind to, ex: :8080, env: WEB_ADDR")

func NewWeb() *WebConfig {
	return &WebConfig{
		Server: &http.Server{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
	}
}

func (w *WebConfig) Validate() error {
	if *webAddr != "" {
		w.Server.Addr = *webAddr
	}
	if w.Server.Addr == "" {
		return ErrNoWebAddr
	}
	return nil
}
