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
	"errors"

	"github.com/go-redis/redis/v7"
)

var (
	ErrNoRedisAddr    = errors.New("missing Redis addr")
	ErrInvalidEncrypt = errors.New("invalid encryption")
	ErrNoEncryptKey   = errors.New("missing EncryptionKey")
)

// RedisConfig holds config for Redis and encryption
type RedisConfig struct {
	// ClientConfig uses https://godoc.org/github.com/go-redis/redis#NewUniversalClient
	ClientConfig  *redis.UniversalOptions
	EncryptionKey []string
	Encryption    int
}

func NewRedis() *RedisConfig {
	return &RedisConfig{ClientConfig: &redis.UniversalOptions{}}
}

func (c *RedisConfig) Validate() error {
	if len(c.ClientConfig.Addrs) < 1 {
		return ErrNoRedisAddr
	}
	if c.Encryption < 0 || c.Encryption > 1 {
		return ErrInvalidEncrypt
	}
	if len(c.EncryptionKey) != 1 { // only allow 1 key for now
		return ErrNoEncryptKey
	}
	return nil
}
