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

package findx

import (
	"errors"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds metrics info
type Metrics struct {
	Keyspace      string
	PromNamespace string
	PromReg       prometheus.Registerer
	addSuc        atomic.Uint64 // added to chan
	addFail       atomic.Uint64 // chan full
	addRej        atomic.Uint64 // reject adding to chan
	sentSuc       atomic.Uint64 // sent to findX successfully
	sentFail      atomic.Uint64 // sent to findX failed
	sentRej       atomic.Uint64 // received 4xx from findX
}

func (m *Metrics) initPrometheus() error {
	if m.PromReg == nil {
		m.PromReg = prometheus.DefaultRegisterer
	}
	if m.Keyspace == "" {
		return errors.New("missing keyspace")
	}
	if m.PromNamespace == "" {
		return errors.New("missing PromNamespace")
	}
	// Not using CounterVec because we want to track metrics independant of metrics Provider
	counters := []struct {
		name    string
		code    string
		counter *atomic.Uint64
	}{
		{"add", "suc", &m.addSuc},
		{"add", "fail", &m.addFail},
		{"add", "rej", &m.addRej},
		{"sent", "suc", &m.sentSuc},
		{"sent", "fail", &m.sentFail},
		{"sent", "rej", &m.sentRej},
	}
	for _, c := range counters {
		c := c
		err := m.PromReg.Register(
			prometheus.NewCounterFunc(
				prometheus.CounterOpts{
					Namespace:   m.PromNamespace,
					Subsystem:   "findx",
					Name:        c.name,
					Help:        "A counter for total number of requests " + c.name + " to FindX",
					ConstLabels: prometheus.Labels{"keyspace": m.Keyspace, "code": c.code},
				},
				func() float64 { return float64(c.counter.Load()) }),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddSuc increments addSuc
func (m *Metrics) AddSuc() { m.addSuc.Add(1) }

// AddFail increments addFail
func (m *Metrics) AddFail() { m.addFail.Add(1) }

// AddRej increments addReject
func (m *Metrics) AddRej() { m.addRej.Add(1) }

// SentSuc increments sentSuc
func (m *Metrics) SentSuc() { m.sentSuc.Add(1) }

// SentFail increments sentFail
func (m *Metrics) SentFail() { m.sentFail.Add(1) }

// SentRej increments sentRej, when receive 4xx (mostly should be 429)
func (m *Metrics) SentRej() { m.sentRej.Add(1) }
