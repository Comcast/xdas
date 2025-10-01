/*
 * Copyright 2025 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
