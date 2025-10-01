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

package main

import (
	"net/http"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type appMetrics struct {
	counter  *prometheus.CounterVec
	duration *prometheus.HistogramVec
	// responseSize  *prometheus.HistogramVec
	// requestSize   *prometheus.HistogramVec
	redisReadErr  prometheus.Counter
	redisWriteErr prometheus.Counter
}

func newMetrics() *appMetrics {
	metrics := &appMetrics{
		counter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_requests_total",
				Help: "A counter for total number of requests.",
			},
			[]string{"app", "code", "method", "keyspace", "client"}, // app name, status code, http method, request URL
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_request_duration_seconds",
				Help:    "A histogram of latencies for requests.",
				Buckets: []float64{.001, .01, .03, 0.1, 0.5, 1, 3, 10, 130},
			},
			[]string{"app"},
		),
		// requestSize: prometheus.NewHistogramVec(
		// 	prometheus.HistogramOpts{
		// 		Name:    "api_request_size_bytes",
		// 		Help:    "A histogram of request sizes for requests.",
		// 		Buckets: []float64{200, 500, 1000, 10000, 100000},
		// 	},
		// 	[]string{"app", "keyspace"},
		// ),
		// responseSize: prometheus.NewHistogramVec(
		// 	prometheus.HistogramOpts{
		// 		Name:    "api_response_size_bytes",
		// 		Help:    "A histogram of response sizes for requests.",
		// 		Buckets: []float64{200, 500, 1000, 10000, 100000},
		// 	},
		// 	[]string{"app", "keyspace"},
		// ),
		redisReadErr: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   AppName,
				Name:        "redis_errors_total",
				Help:        "A counter of Redis errors.",
				ConstLabels: prometheus.Labels{"ops": "read"},
			},
		),
		redisWriteErr: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   AppName,
				Name:        "redis_errors_total",
				Help:        "A counter of Redis errors.",
				ConstLabels: prometheus.Labels{"ops": "write"},
			},
		),
	}
	// prometheus.MustRegister(metrics.counter, metrics.duration, metrics.responseSize, metrics.requestSize,
	prometheus.MustRegister(metrics.counter, metrics.duration, metrics.redisReadErr, metrics.redisWriteErr)
	createBuildInfoMetrics()
	return metrics
}

func createBuildInfoMetrics() {
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: AppName,
			Name:      "build_info",
			Help:      "Build Information",
		},
		[]string{"buildtime", "goversion", "version"},
	)
	prometheus.MustRegister(buildInfo)
	buildInfo.With(prometheus.Labels{"buildtime": BuildTime, "goversion": runtime.Version(), "version": AppVersion}).Set(1)
}

func (m *appMetrics) appMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyspace := chi.URLParam(r, "keyspace")
		const maxUA = 12
		ua := strings.Split(r.UserAgent(), "/")[0]
		if len(ua) > maxUA {
			ua = ua[:maxUA]
		}

		promhttp.InstrumentHandlerDuration(m.duration.MustCurryWith(prometheus.Labels{"app": AppName}),
			promhttp.InstrumentHandlerCounter(m.counter.MustCurryWith(prometheus.Labels{"app": AppName, "keyspace": keyspace, "client": ua}), next),
			// promhttp.InstrumentHandlerCounter(m.counter.MustCurryWith(prometheus.Labels{"app": AppName, "keyspace": keyspace, "client": ua}),
			// 	promhttp.InstrumentHandlerRequestSize(m.requestSize.MustCurryWith(prometheus.Labels{"app": AppName, "keyspace": keyspace}),
			// 		promhttp.InstrumentHandlerResponseSize(m.responseSize.MustCurryWith(prometheus.Labels{"app": AppName, "keyspace": keyspace}), next),
			// 	),
			// ),
		).ServeHTTP(w, r)
	})
}
