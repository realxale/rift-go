package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func RedisConnInit() error {
	redis_password := os.Getenv("REDIS_PASSWORD")
	redis_addr := os.Getenv("REDIS_ADRESS")

	Rdb = redis.NewClient(&redis.Options{
		Addr:     redis_addr,
		Password: redis_password,
		DB:       0,
	})

	ctx := context.Background()
	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}
