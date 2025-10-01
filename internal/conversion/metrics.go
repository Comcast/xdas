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

package conversion

import (
	"errors"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// Using different implementation than findx to avoid map lookup if metrics not set

type metricsProvider interface {
	init() error
	incContentEncodingSuc(keyspace string)
	incContentEncodingFail(keyspace string)
	incContentTypeSuc(keyspace string)
	incContentTypeFail(keyspace string)
	incEncryptionSuc(keyspace string)
	incEncryptionFail(keyspace string)
}

type prometheusMetrics struct {
	PromReg       prometheus.Registerer
	PromNamespace string
	Keyspaces     []string
	counters      map[string]*counterType
	unknown       *counterType
}

type counterType struct {
	contentEncodingSuc  uint64
	contentEncodingFail uint64
	contentTypeSuc      uint64
	contentTypeFail     uint64
	encryptionSuc       uint64
	encryptionFail      uint64
}

func (p *prometheusMetrics) init() error {
	if p.PromReg == nil || p.PromNamespace == "" {
		return errors.New("Missing reg or namespace")
	}
	p.counters = make(map[string]*counterType)
	for _, keyspace := range p.Keyspaces {
		counterType := &counterType{}
		p.counters[keyspace] = counterType

		counters := []struct {
			name    string
			help    string
			code    string
			counter *uint64
		}{
			{"ce", "content-encoding", "suc", &counterType.contentEncodingSuc},
			{"ce", "content-encoding", "fail", &counterType.contentEncodingFail},
			{"ct", "content-type", "suc", &counterType.contentTypeSuc},
			{"ct", "content-type", "fail", &counterType.contentTypeFail},
			{"en", "encryption", "suc", &counterType.encryptionSuc},
			{"en", "encryption", "fail", &counterType.encryptionFail},
		}
		for _, c := range counters {
			c := c
			err := p.PromReg.Register(
				prometheus.NewCounterFunc(
					prometheus.CounterOpts{
						Namespace:   p.PromNamespace,
						Subsystem:   "convert",
						Name:        c.name,
						Help:        "A counter for total number of conversion for " + c.help,
						ConstLabels: prometheus.Labels{"keyspace": keyspace, "code": c.code},
					},
					func() float64 { return float64(atomic.LoadUint64(c.counter)) }),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *prometheusMetrics) incContentEncodingSuc(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.contentEncodingSuc, 1)
	} else {
		atomic.AddUint64(&p.unknown.contentEncodingSuc, 1)
	}
}

func (p *prometheusMetrics) incContentEncodingFail(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.contentEncodingFail, 1)
	} else {
		atomic.AddUint64(&p.unknown.contentEncodingFail, 1)
	}
}

func (p *prometheusMetrics) incContentTypeSuc(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.contentTypeSuc, 1)
	} else {
		atomic.AddUint64(&p.unknown.contentTypeSuc, 1)
	}
}

func (p *prometheusMetrics) incContentTypeFail(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.contentTypeFail, 1)
	} else {
		atomic.AddUint64(&p.unknown.contentTypeFail, 1)
	}
}

func (p *prometheusMetrics) incEncryptionSuc(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.encryptionSuc, 1)
	} else {
		atomic.AddUint64(&p.unknown.encryptionSuc, 1)
	}
}

func (p *prometheusMetrics) incEncryptionFail(keyspace string) {
	counter := p.counters[keyspace]
	if counter != nil {
		atomic.AddUint64(&counter.encryptionFail, 1)
	} else {
		atomic.AddUint64(&p.unknown.encryptionFail, 1)
	}
}

type noMetrics struct{}

func (n *noMetrics) init() error                            { return nil }
func (n *noMetrics) incContentEncodingSuc(keyspace string)  {}
func (n *noMetrics) incContentEncodingFail(keyspace string) {}
func (n *noMetrics) incContentTypeSuc(keyspace string)      {}
func (n *noMetrics) incContentTypeFail(keyspace string)     {}
func (n *noMetrics) incEncryptionSuc(keyspace string)       {}
func (n *noMetrics) incEncryptionFail(keyspace string)      {}
