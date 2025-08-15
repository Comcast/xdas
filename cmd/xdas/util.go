// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"bytes"
	"io"
	"net/url"
	"strconv"
	"sync"
	"time"
	"xdas/internal/magicbyte"
)

// readAll reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with a specified capacity.
// copied from ioutil so pool.sync can be used
func readAll(r io.Reader, buf *bytes.Buffer) (b []byte, err error) {
	//	var buf bytes.Buffer
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	buf.Grow(bytes.MinRead)
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}

// MaxRetryGet is how many times we will retry HTTP GET
const MaxRetryGet = 3

/*
func retryGet(c *http.Client, url string) (*http.Response, error) {
	for retries := 0; retries < MaxRetryGet; retries++ {
		resp, err := c.Get(url)
		if err != nil || resp.StatusCode >= 500 {
			if err == nil {
				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
			}
			sleepTime := math.Pow(2, float64(retries)) * float64(500*time.Millisecond)
			time.Sleep(time.Duration(sleepTime))
		} else {
			return resp, err
		}
	}
	return nil, errors.New("")
}
*/

func writeToBufPool(pool *sync.Pool, mb byte, data []byte) (b *bytes.Buffer) {
	b = pool.Get().(*bytes.Buffer)
	b.Reset()
	b.Grow(magicbyte.MagicByteLength + len(data))
	b.WriteByte(mb)
	b.Write(data)
	return
}

// getTTL returns the TTL used for Redis set. t option can be either with or without time unit.
// If no time unit is specified, second is assumed. Valid time units are "ns", "us" (or "µs"),
// "ms", "s", "m", "h". If the t option is not valid, t2 will be used, if t2 is not set,
// defaultGlobalTTL will be used.
func getTTL(t string, t2 time.Duration) (ttl time.Duration) {
	if t != "" {
		i, err := strconv.Atoi(t)
		if err == nil {
			return time.Duration(i) * time.Second
		}
		ttl, err := time.ParseDuration(t)
		if err == nil {
			return ttl
		}
	}
	if t2 > 0 {
		ttl = t2
	} else {
		ttl = defaultGlobalTTL
	}
	return ttl
}

// parseBool returns true if key is present URL without any value, or if the value is set to true.
// Otherwise it returns false.
func parseBool(key string, v url.Values) bool {
	if !v.Has(key) {
		return false
	}

	value := v.Get(key)
	if value == "" {
		return true
	}
	result, _ := strconv.ParseBool(value)
	return result
}
