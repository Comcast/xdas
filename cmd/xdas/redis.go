// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"errors"
	"xdas/internal/magicbyte"

	"github.com/go-redis/redis/v7"
)

func redisGet(rClient redis.UniversalClient, key string) (magicbyte.MagicByte, []byte, error) {
	result, err := rClient.Get(key).Bytes()
	if err != nil {
		return magicbyte.MagicByte{}, result, err
	}
	return redisParseResult(result)
}

// func redisHGet(rClient redis.UniversalClient, key, field string) (magicbyte.MagicByte, []byte, error) {
// 	result, err := rClient.HGet(key, field).Bytes()
// 	if err != nil {
// 		return magicbyte.MagicByte{}, result, err
// 	}
// 	return redisParseResult(result)
// }

func redisParseResult(input []byte) (magicByte magicbyte.MagicByte, result []byte, err error) {
	// currently all keyspace must have magicByte. will have keyspace without magicByte in the future for atomic operation
	if len(input) < magicbyte.MagicByteLength { // need better handling, not normal with magicByte
		return magicByte, result, errors.New("redis Get error, doesn't have magicByte")
	}
	magicByte = magicbyte.NewFrom(input[0])
	result = input[magicbyte.MagicByteLength:]
	return
}

// redisKeyCT construct the key used for Cujo Threat Notification for Redis Cluster
// in the form of <keyspace>:{<id>}_<epochHour>_<quarter>
func redisKeyCT(keyspace, id, epochHour, quarter string) string {
	return keyspace + ":{" + id + "}" + "_" + epochHour + "_" + quarter
}

// redisKey construct the key used for Redis Cluster in the form of <keyspace>:{<id>}
func redisKey(keyspace, id string) string { return keyspace + ":{" + id + "}" }

/*
func redisKeyDay(keyspace, id, day string) string {
	return keyspace + ":{" + id + "}" + "_" + day
}
*/
