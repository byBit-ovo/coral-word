package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
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

// func (client *RedisClient) SetUserBook(userId, bookName, bookId string) error {
// 	key := userId + "_" + bookName
// 	return client.client.HSet(context.Background(), "coral_word_user_book",key,bookId).Err()
// }
// func (client *RedisClient) GetUserBookId(userId, bookName string) (string, error) {
// 	key := userId + "_" + bookName
// 	return client.client.HGet(context.Background(), "coral_word_user_book",key).Result()
// }
// func (client *RedisClient) DelUserBookId(userId, bookName string) error {
// 	key := userId + "_" + bookName
// 	return client.client.HDel(context.Background(), "coral_word_user_book", key).Err()
// }

func (client *RedisClient) SetQueryingWord(words ...string) error {
	querying := "querying_"
	for _, word := range words {
		client.client.Set(context.Background(), querying+word, "1", 5*time.Minute)
	}
	return nil
}

func (client *RedisClient) IsQueryingWord(word string) (bool, error) {
	res, err := client.client.Get(context.Background(), "querying_"+word).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return res == "1", nil
}

func (client *RedisClient) DelQueryingWord(words ...string) error {
	for _, word := range words {
		if err := client.client.Del(context.Background(), "querying_"+word).Err(); err != nil {
			return err
		}
	}
	return nil
}

// ReviewSession 存储到 Redis（30 分钟过期）
const reviewSessionTTL = 30 * time.Minute
const reviewSessionPrefix = "review_session:"

func (client *RedisClient) SetReviewSession(sessionID string, session *ReviewSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return client.client.Set(context.Background(), reviewSessionPrefix+sessionID, data, reviewSessionTTL).Err()
}

func (client *RedisClient) GetReviewSession(sessionID string) (*ReviewSession, error) {
	key := reviewSessionPrefix + sessionID
	log.Println("GetReviewSession", key)
	data, err := client.client.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, nil // session 不存在
	}
	if err != nil {
		return nil, err
	}
	var session ReviewSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (client *RedisClient) DelReviewSession(sessionID string) error {
	return client.client.Del(context.Background(), reviewSessionPrefix+sessionID).Err()
}




