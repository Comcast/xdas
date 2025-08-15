// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v7"
)

// handleFuncXdasAtomicGet returns the value of an atomic keyspace
func (s *Server) handleFuncXdasAtomicGet(key string, w http.ResponseWriter, r *http.Request) {
	result, err := s.redis.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
			return
		}
		s.sendRedisReadErr(w, err)
		return
	}
	// w.Header().Set("Content-type", "application/octet-stream")
	w.Write(result)
}

// handleFuncXdasAtomicInc increments value of an atomic keyspace by the query parameter n or 1.
func (s *Server) handleFuncXdasAtomicInc(w http.ResponseWriter, r *http.Request) {
	keyspace := chi.URLParam(r, "keyspace")
	id := getID(r)
	key := redisKey(keyspace, id)
	ksConf, ok := s.config.Keyspaces[keyspace]
	if !ok { // should not happen, already checked by validateAtomicKeyspace
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	ttl := getTTL(r.Header.Get("Xttl"), ksConf.ttl)
	n, _ := strconv.ParseInt(r.URL.Query().Get("n"), 10, 0)
	if n == 0 {
		n = 1
	}

	s.atomicIncrBy(key, n, ttl, w)
}

func (s *Server) atomicIncrBy(key string, n int64, ttl time.Duration, w http.ResponseWriter) {
	pipe := s.redis.Pipeline()
	result := pipe.IncrBy(key, n)
	pipe.Expire(key, ttl)
	_, err := pipe.Exec()
	if err != nil {
		s.sendRedisWriteErr(w, err)
		return
	}
	fmt.Fprint(w, result.Val())
}
