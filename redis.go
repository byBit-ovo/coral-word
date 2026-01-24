package main

import (
	"context"
	"log"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
)

var redisClient *redis.Client
type RedisWordClient struct {
	client *redis.Client
}

var redisWordClient *RedisWordClient
func InitRedis() error{
	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "",
		DB:       0,
	})
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis connection error:", err)
		return err
	}
	redisWordClient = &RedisWordClient{client: redisClient}
	return nil
}

// word: wordId
func (client *RedisWordClient) HSet(word string, id int64) error {
	return client.client.HSet(context.Background(), "coral_word", word, id).Err()
}

// word: wordId
func (client *RedisWordClient) HGet(word string) (int64, error) {
	res, err := client.client.HGet(context.Background(), "coral_word", word).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(res, 10, 64)
}

// word: wordId
func (client *RedisWordClient) HGetAll() (map[string]string, error) {
	return client.client.HGetAll(context.Background(), "coral_word").Result()
}
func (client *RedisWordClient) HDel(word string) error {
	return client.client.HDel(context.Background(), "coral_word", word).Err()
}

func (client *RedisWordClient) HLen() (int64, error) {
	return client.client.HLen(context.Background(), "coral_word").Result()
}
