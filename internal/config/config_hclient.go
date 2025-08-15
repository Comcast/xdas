// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

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
