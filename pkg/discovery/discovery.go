package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"gateway/pkg/configs"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"runtime/debug"
	"time"

	"github.com/redis/go-redis/v9"
)

func RegisterSelf() error {
	if configs.GetDiscovery().Type == "redis" {
		return registerSelfToRedis()
	}

	return nil
}

func registerSelfToRedis() error {
	oldRedisAddress := ""
	oldPassword := ""
	var rdb *redis.Client = nil

	writeToRedis := func() error {
		discoveryConfig := configs.GetDiscovery()

		// Password updated, reconnecting
		if oldRedisAddress != discoveryConfig.RedisAddress || oldPassword != discoveryConfig.RedisPassword {
			if rdb != nil {
				rdb.Close()
			}

			rdb = redis.NewClient(&redis.Options{
				Addr:         discoveryConfig.RedisAddress,
				Password:     discoveryConfig.RedisPassword,
				DB:           0,
				DialTimeout:  7 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
			})

			oldRedisAddress = discoveryConfig.RedisAddress
			oldPassword = discoveryConfig.RedisPassword
		}

		if rdb != nil {
			nodeInfoConfig := configs.GetNodeInfo()
			// Online users
			nodeInfoConfig.ConnectionNum = uint64(metric.CountConnection.Load())
			nodeInfoConfig.MetricData = metric.Data
			// Expiration time
			nodeInfoConfig.ExpireTime = uint64(time.Now().Add(10 * time.Second).Unix())
			data, err := json.Marshal(nodeInfoConfig)
			if err != nil {
				return err
			}

			pipe := rdb.Pipeline()
			pipe.HSet(context.TODO(), discoveryConfig.RedisRegisterKey, nodeInfoConfig.ID, string(data))
			pipe.Expire(context.TODO(), discoveryConfig.RedisRegisterKey, 86400*3*time.Second)
			_, err = pipe.Exec(context.TODO())
			if err != nil && err != redis.Nil {
				return err
			}
		}

		return nil
	}

	// First-time registration
	if err := writeToRedis(); err != nil {
		return err
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				utils.AlertAuto(fmt.Sprintf("discovery register panic: %v stack: %s", err, string(debug.Stack())))
			}
		}()

		for {
			time.Sleep(5 * time.Second)
			if err := writeToRedis(); err != nil {
				utils.AlertAuto("discovery register to redis fail: " + err.Error())
			}
		}
	}()

	return nil
}
