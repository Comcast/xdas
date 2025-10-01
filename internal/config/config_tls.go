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
	"crypto/x509"
	"errors"
	"os"
)

// TLS holds config for TLS
type TLS struct {
	CertFile string // Client or Server cert
	KeyFile  string
	CaFile   string // Root cert
	Insecure bool
	cert     []tls.Certificate
	ca       *x509.CertPool
}

// GetClientTLS return tls.config for use by client. Invoke Validate before calling this.
func (t *TLS) GetClientTLS() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: t.Insecure,
		Certificates:       t.cert,
		RootCAs:            t.ca,
	}
}

// GetServerTLS return tls.config for use by server. Invoke Validate before calling this.
func (t *TLS) GetServerTLS() *tls.Config {
	if t.cert == nil {
		return nil
	}

	c := &tls.Config{Certificates: t.cert}
	if t.ca != nil {
		c.ClientAuth = tls.RequireAndVerifyClientCert
		c.ClientCAs = t.ca
	}
	return c
}

// Set tls.config based on config. Invoke Validate before calling this.
func (t *TLS) Set(orig *tls.Config) {
	orig.Certificates = t.cert
	orig.RootCAs = t.ca
	orig.InsecureSkipVerify = t.Insecure
}

func (t *TLS) Validate() error {
	if t.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return err
		}
		t.cert = []tls.Certificate{cert}
	}
	if t.CaFile != "" {
		caCert, err := os.ReadFile(t.CaFile)
		if err != nil {
			return err
		}
		caCertPool := x509.NewCertPool()
		if caCertPool.AppendCertsFromPEM(caCert) {
			t.ca = caCertPool
		} else {
			return errors.New("unable to load CA")
		}
	}
	return nil
}
