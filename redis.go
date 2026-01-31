package main

import (
	"context"
	"log"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
)

var redisClientBase *redis.Client

type RedisClient struct {
	client *redis.Client
}

var redisClient *RedisClient
func InitRedis() error{
	redisClientBase = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "",
		DB:       0,
	})
	_, err := redisClientBase.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis connection error:", err)
		return err
	}
	redisClient = &RedisClient{client: redisClientBase}
	return nil
}

// word: wordId
func (client *RedisClient) HSetWord(word string, id int64) error {
	return client.client.HSet(context.Background(), "coral_word", word, id).Err()
}

// word: wordId
func (client *RedisClient) HGetWord(word string) (int64, error) {
	res, err := client.client.HGet(context.Background(), "coral_word", word).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(res, 10, 64)
}

// word: wordId
func (client *RedisClient) HGetAllWords() (map[string]string, error) {
	return client.client.HGetAll(context.Background(), "coral_word").Result()
}
func (client *RedisClient) HDelWord(word string) error {
	return client.client.HDel(context.Background(), "coral_word", word).Err()
}

func (client *RedisClient) HLenWords() (int64, error) {
	return client.client.HLen(context.Background(), "coral_word").Result()
}

func (client *RedisClient) SetUserSession(sessionId string, userId string) error {
	return client.client.HSet(context.Background(), "coral_word_session",sessionId, userId).Err()
}
func (client *RedisClient) GetUserSession(sessionId string) (string, error) {
	return client.client.HGet(context.Background(), "coral_word_session",sessionId).Result()
}
func (client *RedisClient) DelUserSession(sessionId string) error {
	return client.client.HDel(context.Background(), "coral_word_session", sessionId).Err()
}

func (client *RedisClient) SetUserName(userId string, userName string) error {
	return client.client.HSet(context.Background(), "coral_word_user",userId,userName).Err()
}
func (client *RedisClient) GetUserName(userId string) (string, error) {
	return client.client.HGet(context.Background(), "coral_word_user",userId).Result()
}
func (client *RedisClient) DelUserName(userId string) error {
	return client.client.HDel(context.Background(), "coral_word_user", userId).Err()
}


