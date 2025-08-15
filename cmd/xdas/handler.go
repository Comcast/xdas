// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"xdas/internal/conversion"
	"xdas/internal/keyspaces"
	"xdas/internal/magicbyte"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v7"
)

func (s *Server) handleFuncXdasGet(w http.ResponseWriter, r *http.Request) {
	keyspace := chi.URLParam(r, "keyspace")
	id := getID(r)
	key := redisKey(keyspace, id)
	s.xdasCommonGet(keyspace, id, key, w, r)
}

func (s *Server) xdasCommonGet(keyspace, id, key string, w http.ResponseWriter, r *http.Request) {
	ksConf, ok := s.config.Keyspaces[keyspace]
	if !ok { // should not happen, handled by validateKeyspace
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if ksConf.Kind == keyspaces.KSAtomic { // Atomic keyspaces are native Redis string type
		s.handleFuncXdasAtomicGet(key, w, r)
		return
	}

	magicByte, data, err := redisGet(s.redis, key)
	if err != nil {
		if err == redis.Nil {
			if !parseBool("nofindx", r.URL.Query()) {
				findX(s.redis, ksConf, keyspace, id)
			}
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
			return
		}
		s.sendRedisReadErr(w, err)

		return
	}

	var outMagicByte magicbyte.MagicByte
	switch outFormat := r.URL.Query().Get("format"); outFormat {
	case "":
		outMagicByte = ksConf.Output.magicByte
	case "raw":
		w.Header().Set("Content-type", "application/octet-stream")
		w.Write([]byte{magicByte.Get()})
		w.Write(data)
		return
	default:
		outMagicByte = magicbyte.New("", outFormat, 0)
		if outMagicByte.GetCTV() == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	magicByte, data, err = conversion.Convert(keyspace, magicByte, outMagicByte, data)
	if err != nil {
		s.log.Error("Conversion error", "keyspace", keyspace, "key", key, "err", err)
		// return error or original content?
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	magicByte.SetContentHeaders(w.Header())
	w.Header().Set("Content-length", strconv.Itoa(len(data)))
	w.Write(data)
}

// func (s *Server) handleFuncXdasRawGet(w http.ResponseWriter, r *http.Request) {
// 	keyspace := chi.URLParam(r, "keyspace")
// 	id := strings.ToUpper(chi.URLParam(r, "id"))
// 	key := redisKey(keyspace, id)

// 	result, err := s.redis.Get(key).Bytes()
// 	if err != nil {
// 		if err == redis.Nil {
// 			w.WriteHeader(http.StatusNotFound)
// 			fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
// 			return
// 		}
// 		s.Println("Redis Get error:", err)
// 		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
// 		return
// 	}
// 	// w.Header().Set("Content-type", "application/octet-stream")
// 	w.Write(result)
// }

func (s *Server) handleFuncXdasPut(w http.ResponseWriter, r *http.Request) {
	keyspace := chi.URLParam(r, "keyspace")
	ksConf, ok := s.config.Keyspaces[keyspace]
	if !ok {
		// s.Println("Invalid keyspace", keyspace)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id := getID(r)
	key := redisKey(keyspace, id)

	magicByte := magicbyte.New(r.Header.Get("Content-encoding"), r.Header.Get("Content-type"), 0)
	inputMagicByte := ksConf.Input.magicByte

	// validate content-type and content-encoding against config
	if inputMagicByte.GetCEV() != 0 && inputMagicByte.GetCEV() != magicByte.GetCEV() ||
		inputMagicByte.GetCTV() != 0 && inputMagicByte.GetCTV() != magicByte.GetCTV() {
		s.log.Info("Invalid content format", "keyspace", keyspace, "id", id, "mb", magicByte)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	b1 := s.bufPool.Get().(*bytes.Buffer)
	defer s.bufPool.Put(b1)
	b1.Reset()
	data, err := readAll(r.Body, b1)
	if err != nil {
		s.sendRequestBodyReadErr(w, err)
		return
	}

	if s.config.ValidateContent {
		_, err := conversion.Unpack(keyspace, magicByte, data)
		if err != nil {
			s.log.Info("Invalid request", "keyspace", keyspace, "id", id, "err", err, "data", data)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	storeMagicByte := ksConf.Store.magicByte

	// may remove this validation in the future
	if storeMagicByte.GetCEV() == 0 && magicByte.GetCEV() != 0 {
		_, err := conversion.Unpack(keyspace, magicByte, data)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	magicByte, data, err = conversion.Convert(keyspace, magicByte, storeMagicByte, data)
	if err != nil {
		s.log.Error("Conversion error", "keyspace", keyspace, "key", key, "err", err)
		// Most likely bad content or encoding, should return BadRequest in the future
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ttl := getTTL(r.Header.Get("Xttl"), ksConf.ttl)

	b2 := writeToBufPool(&s.bufPool, magicByte.Get(), data)
	defer s.bufPool.Put(b2)

	result, err := s.redis.Set(key, b2.Bytes(), ttl).Result()
	if err != nil { // set MaxRetries under Redis:ClientConfig in config to retry
		s.sendRedisWriteErr(w, err)
		return
	}
	fmt.Fprintln(w, result)
}

func (s *Server) handleFuncXdasMultiGet(w http.ResponseWriter, r *http.Request) {
	id := getID(r)
	var reqKeyspaces []string
	if ks := r.URL.Query().Get("ks"); ks == "" {
		reqKeyspaces = s.config.Multipart.Keyspaces
	} else {
		reqKeyspaces = strings.Split(ks, ",")
	}
	keys := make([]string, len(reqKeyspaces))
	keyspaces := make([]string, len(reqKeyspaces))
	ksConfs := make([]*KeyspaceConfig, len(reqKeyspaces))
	var validKeyspaceCount int
	for _, reqKeyspace := range reqKeyspaces {
		ksConf, ok := s.config.Keyspaces[reqKeyspace]
		if !ok {
			s.log.Info("Invalid keyspace in multi", "keyspace", reqKeyspace, "ip", r.RemoteAddr, "url", r.RequestURI)
			continue
		}
		if reqKeyspace != "ct" {
			keys[validKeyspaceCount] = redisKey(reqKeyspace, id)
		} else {
			epochHour := r.URL.Query().Get("ct_hour")
			quarter := r.URL.Query().Get("ct_quarter")
			keys[validKeyspaceCount] = redisKeyCT(reqKeyspace, id, epochHour, quarter)
		}
		keyspaces[validKeyspaceCount] = reqKeyspace
		ksConfs[validKeyspaceCount] = ksConf
		validKeyspaceCount++
	}
	keys = keys[:validKeyspaceCount]
	keyspaces = keyspaces[:validKeyspaceCount]
	ksConfs = ksConfs[:validKeyspaceCount]
	if len(keyspaces) < 1 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
		return
	}

	results, err := s.redis.MGet(keys...).Result()
	if err != nil {
		s.sendRedisReadErr(w, err)
		return
	}

	// b1 := s.bufPool.Get().(*bytes.Buffer)
	// defer s.bufPool.Put(b1)
	// b1.Reset()
	// mw := multipart.NewWriter(b1)
	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())
	var validResultCount int
	for index, result := range results {
		if result == nil {
			if !parseBool("nofindx", r.URL.Query()) {
				findX(s.redis, ksConfs[index], keyspaces[index], id)
			}
			continue
		}
		r, ok := result.(string)
		if !ok || len(r) < magicbyte.MagicByteLength {
			continue
		}
		magicByte := magicbyte.NewFrom(r[0])
		data := []byte(r[magicbyte.MagicByteLength:])

		outMagicByte := ksConfs[index].Output.magicByte
		magicByte, data, err = conversion.Convert(keyspaces[index], magicByte, outMagicByte, data)
		if err != nil {
			s.log.Error("Data conversion error:", "keyspace", keyspaces[index], "key", keys[index], "err", err)
			continue
		}

		h := make(textproto.MIMEHeader)
		magicByte.SetContentHeaders(h)
		if ks := keyspaces[index]; ks != "ct" {
			h.Set("Namespace", keyspaces[index])
		} else {
			keyParts := strings.Split(keys[index], "_")
			if len(keyParts) != 3 {
				s.log.Error("Invalid ct key", "key", keys[index])
				continue
			}
			h.Set("Namespace", ks+"_"+keyParts[1]+"_"+keyParts[2])
		}
		part, err := mw.CreatePart(h)
		if err != nil {
			s.log.Error("Multipart creation error:", "key", keys[index], "err", err)
			continue
		}
		part.Write(data)
		validResultCount++
	}
	if validResultCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
		return
	}
	mw.Close()
	// w.Header().Set("Content-length", strconv.Itoa(b1.Len()))
	// b1.WriteTo(w)
	// w.Write(b1.Bytes())
}

// No app is using this, to be implemented in the future
// func (s *Server) handleFuncXdasMultiPut(w http.ResponseWriter, r *http.Request) {}

func (s *Server) handleFuncXdasDel(w http.ResponseWriter, r *http.Request) {
	keyspace := chi.URLParam(r, "keyspace")
	id := getID(r)
	key := redisKey(keyspace, id)

	result, err := s.redis.Del(key).Result()
	if err != nil {
		s.sendRedisWriteErr(w, err)
		return
	}
	if result < 1 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
		return
	}
	fmt.Fprintln(w, result)
}

func (s *Server) sendRequestBodyReadErr(w http.ResponseWriter, err error) {
	s.log.Info("Error reading body", "err", err)

	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func (s *Server) sendRedisReadErr(w http.ResponseWriter, err error) {
	s.log.Error("Redis read error", "err", err)
	s.metrics.redisReadErr.Inc()
	http.Error(w, "Internal Server Error 10", http.StatusInternalServerError)
}

func (s *Server) sendRedisWriteErr(w http.ResponseWriter, err error) {
	s.log.Error("Redis write error", "err", err)
	s.metrics.redisWriteErr.Inc()
	http.Error(w, "Internal Server Error 11", http.StatusInternalServerError)
}

// By request to print raw config, not a good idea as it contains password
// func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write(s.config.raw)
// }

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := struct {
		Data struct {
			Version   string
			BuildTime string
		} `json:"data"`
	}{
		Data: struct {
			Version   string
			BuildTime string
		}{
			Version:   AppName + "-" + AppVersion,
			BuildTime: BuildTime,
		},
	}
	output, _ := json.Marshal(version)
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, http.StatusText(http.StatusOK))
}

// getID returns the URL parameter value of "id"
func getID(r *http.Request) string {
	return strings.ToUpper(chi.URLParam(r, "id"))
}

func findX(rdb redis.UniversalClient, ksConf *KeyspaceConfig, keyspace, id string) {
	switch keyspace {
	case "pld":
		go func() {
			if rdb.Exists(redisKey("pa", id)).Val() == 0 { // Only trigger findX if id exist in pa keyspace
				ksConf.FindX.Reject()
				return
			}
			ksConf.FindX.Add(id)
		}()
	default:
		ksConf.FindX.Add(id)
	}
}
