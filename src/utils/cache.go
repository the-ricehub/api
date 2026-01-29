package utils

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const testKey = "connTest"

var rdb *redis.Client

func InitCache(connUrl string) {
	logger := zap.L()
	logger.Info("Trying to establish a connection with redis cache...")

	opt, err := redis.ParseURL(connUrl)
	if err != nil {
		logger.Fatal("Failed to parse redis connection URL", zap.Error(err))
	}

	rdb = redis.NewClient(opt)

	// test the connection
	ctx := context.Background()

	// set test data
	testData := strconv.Itoa(rand.IntN(15000))
	err = rdb.Set(ctx, testKey, testData, 5*time.Minute).Err()
	if err != nil {
		logger.Fatal("Failed to set test data in cache", zap.Error(err))
	}

	// retrieve the data
	res, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		logger.Fatal("Failed to retrieve test data from cache", zap.Error(err))
	}
	if res != testData {
		logger.Fatal("Incorrect test data returned from cache",
			zap.String("expected", testData),
			zap.String("got", res),
		)
	}

	logger.Info("Successfully connected to the cache")
}

func CloseCache() {
	rdb.Close()
}

func increment(key string, expireAfter time.Duration) (int64, error) {
	ctx := context.Background()

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		rdb.Expire(ctx, key, expireAfter)
	}

	return count, err
}

func IncrementRateLimit(clientId string, expireAfter time.Duration) (int64, error) {
	key := fmt.Sprintf("rateLimit:%s", clientId)
	return increment(key, expireAfter)
}

func IncrementPathRateLimit(path string, clientId string, expireAfter time.Duration) (int64, error) {
	key := fmt.Sprintf("pathRateLimit:%s-%s", path, clientId)
	return increment(key, expireAfter)
}
