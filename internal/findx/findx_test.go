// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package findx

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestStart(t *testing.T) {
	f1 := &FindX{}
	f2 := &FindX{Enabled: true}
	f3 := &FindX{
		Enabled:           true,
		URL:               "http://test/findx/",
		ChannelBufferSize: 200,
		Thread:            2,
		// HTTPClient:        hClient,
	}
	f4 := &FindX{
		Enabled:           true,
		URL:               "http://test/findx/",
		ChannelBufferSize: -1,
		Thread:            0,
	}
	tests := []struct {
		findX   *FindX
		wantErr bool
	}{
		{f1, true},
		{f2, true},
		{f3, false},
		{f4, false},
	}
	for _, tt := range tests {
		findX := tt.findX
		err := findX.Start()
		if tt.wantErr {
			if err == nil {
				t.Errorf("Start() got nil error, want error\n")
				continue
			} else {
				continue
			}
		}
		if err != nil {
			t.Errorf("Start() returned error: %v\n", err)
			continue
		}
		if !findX.enabled {
			t.Error("FindX.enable want: true, got:", findX.enabled)
		}
		if findX.ChannelBufferSize < 1 {
			t.Errorf("FindX.channel size got %v, want: %v,\n", findX.ChannelBufferSize, DefaultChannelBufferSize)
		}
		if findX.Thread < 1 {
			t.Errorf("FindX.thread got %v, want: %v\n", findX.Thread, DefaultThread)
		}
	}
}

func TestRun(t *testing.T) {
	url := "http://test/findx/"
	key := "test"
	hClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.String() != url+key {
			t.Errorf("Wrong URL got: %v, want: %v\n", req.URL.String(), url+key)
		}
		return &http.Response{
			StatusCode: 202,
			Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
	})
	hClientFail := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
	})
	hClientRej := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
	})
	t.Run("send success", func(t *testing.T) {
		findX := &FindX{
			Enabled:    true,
			URL:        url,
			HTTPClient: hClient,
		}
		findX.Start()
		maxReq := 10
		for i := 0; i < maxReq; i++ {
			findX.Add(key)
		}
		findX.Close()
		if actual := findX.Metrics.sentSuc.Load(); actual != uint64(maxReq) {
			t.Errorf("Incorrect number of sentSuc got: %v, want: %v\n", actual, maxReq)
		}
	})
	t.Run("send fail", func(t *testing.T) {
		findX := &FindX{
			Enabled:    true,
			URL:        url,
			HTTPClient: hClientFail,
		}
		findX.Start()
		maxReq := 10
		for i := 0; i < maxReq; i++ {
			findX.Add(key)
		}
		findX.Close()
		if actual := findX.Metrics.sentFail.Load(); actual != uint64(maxReq) {
			t.Errorf("Incorrect number of failed requests got: %v, want: %v\n", actual, maxReq)
		}
	})
	t.Run("send rej", func(t *testing.T) {
		findX := &FindX{
			Enabled:    true,
			URL:        url,
			HTTPClient: hClientRej,
		}
		findX.Start()
		maxReq := 10
		for i := 0; i < maxReq; i++ {
			findX.Add(key)
		}
		findX.Close()
		if actual := findX.Metrics.sentRej.Load(); actual != uint64(maxReq) {
			t.Errorf("Incorrect number of failed requests got: %v, want: %v\n", actual, maxReq)
		}
	})
}

func TestAdd(t *testing.T) {
	url := "http://test/findx/"
	hClient := NewTestClient(func(req *http.Request) *http.Response {
		time.Sleep(10 * time.Millisecond)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
	})
	t.Run("not enabled", func(t *testing.T) {
		findX := &FindX{
			Enabled:    false,
			URL:        url,
			HTTPClient: hClient,
		}
		findX.Start()
		maxReq := 5
		for i := 0; i < maxReq; i++ {
			findX.Add("test")
		}
		findX.Close()
		if actual := findX.Metrics.addSuc.Load(); actual != 0 {
			t.Errorf("Incorrect number of requests got: %v, want: %v\n", actual, 0)
		}
	})
	t.Run("queue full", func(t *testing.T) {
		size := 2
		findX := &FindX{
			Enabled:           true,
			ChannelBufferSize: size,
			URL:               url,
			HTTPClient:        hClient,
		}
		findX.Start()
		maxReq := 10
		for i := 0; i < maxReq; i++ {
			findX.Add("test")
		}
		findX.Close()
		if actual := findX.Metrics.addSuc.Load(); actual != uint64(size) {
			t.Errorf("Incorrect number of addSuc got: %v, want: %v\n", actual, size)
		}
		if actual := findX.Metrics.addFail.Load(); actual != uint64(maxReq-size) {
			t.Errorf("Incorrect number of addSuc got: %v, want: %v\n", actual, maxReq-size)
		}
	})
}
