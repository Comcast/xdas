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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	DefaultChannelBufferSize = 128
	DefaultThread            = 1
)

// FindX defines parameters to run FindX service.
type FindX struct {
	Enabled           bool
	Keyspace          string
	URL               string
	ChannelBufferSize int
	Thread            int
	HTTPClient        *http.Client
	UserAgent         string
	Metrics           Metrics
	enabled           bool
	ch                chan string
	wg                sync.WaitGroup
}

// Start runs the FindX.
func (f *FindX) Start() error {
	if !f.Enabled {
		return errors.New("FindX not Enabled")
	}
	if _, err := url.ParseRequestURI(f.URL); err != nil {
		f.Enabled = false
		return err
	}
	if err := f.Metrics.initPrometheus(); err != nil {
		// should we panic or ignore
	}
	if f.ChannelBufferSize < 1 {
		f.ChannelBufferSize = DefaultChannelBufferSize
	}
	if f.Thread < 1 {
		f.Thread = DefaultThread
	}
	f.ch = make(chan string, f.ChannelBufferSize)
	f.wg.Add(f.Thread)
	run := f.run
	if f.Keyspace == "dm" {
		run = f.runDM
	}
	for i := 0; i < f.Thread; i++ {
		if f.HTTPClient != nil {
			go run(f.HTTPClient, f.URL)
		} else {
			go run(http.DefaultClient, f.URL)
		}
	}
	f.enabled = true
	return nil
}

// run is default processor for all keyspaces
func (f *FindX) run(hclient *http.Client, URL string) {
	defer f.wg.Done()
	for id := range f.ch {
		url := URL + id
		f.getReq(hclient, url)
	}
}

// RunDM for dm keyspace
func (f *FindX) runDM(hclient *http.Client, URL string) {
	defer f.wg.Done()
	for id := range f.ch {
		ids := strings.Split(id, ",")
		if len(ids) < 2 {
			f.Metrics.SentFail()
			continue
		}
		url := URL + ids[0] + "?devices=" + ids[1]
		f.getReq(hclient, url)
	}
}

func (f *FindX) getReq(hclient *http.Client, url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("create request err", err)
	}
	req.Header.Set("User-Agent", f.UserAgent)

	resp, err := hclient.Do(req)
	if err != nil {
		fmt.Println("findx err", err)
		f.Metrics.SentFail()
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode < 300:
		f.Metrics.SentSuc()
	case resp.StatusCode < 500:
		fmt.Println("findx non-2xx code:", resp.StatusCode, url)
		f.Metrics.SentRej()
	default:
		fmt.Println("findx non-2xx code:", resp.StatusCode, url)
		f.Metrics.SentFail()
	}

	io.Copy(io.Discard, resp.Body) // Ensure keepalive
}

// Add an entry to look up through FindX, it is non-blocking.
func (f *FindX) Add(id string) {
	if !f.enabled {
		return
	}
	select {
	case f.ch <- id:
		f.Metrics.AddSuc()
	default:
		f.Metrics.AddFail()
	}
}

// Reject updates reject FindX metrics
func (f *FindX) Reject() {
	if !f.enabled {
		return
	}
	f.Metrics.AddRej()
}

// Close the FindX service
func (f *FindX) Close() {
	if !f.enabled {
		return
	}
	f.Enabled = false // Not atomic, but okay, closing down anyway
	f.enabled = false
	close(f.ch)
	f.wg.Wait()
}
