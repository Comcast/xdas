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
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"
)

var ErrHClientTransport = errors.New("invalid HTTP transport")

type HClientConfig struct {
	Client *http.Client
	TLS    TLS
}

func NewHClient() *HClientConfig {
	return &HClientConfig{
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          2000,
				MaxIdleConnsPerHost:   1000,
				IdleConnTimeout:       45 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       &tls.Config{},
			},
			Timeout: 5 * time.Second,
		},
	}
}

func (h *HClientConfig) Validate() error {
	if err := h.TLS.Validate(); err != nil {
		return err
	}
	transport, ok := h.Client.Transport.(*http.Transport)
	if !ok {
		return ErrHClientTransport
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = h.TLS.GetClientTLS()
	} else {
		h.TLS.Set(transport.TLSClientConfig)
	}
	return nil
}
