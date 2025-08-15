// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

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
