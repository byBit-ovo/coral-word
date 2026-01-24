package main

import (
	"bufio"
	_ "bytes"
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"log"
	"os"
	_ "strconv"
	"time"

	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Loading env file error")
	}
	if err = llm.InitModels(); err != nil {
		log.Fatal("InitModels error")
		return
	}
	if err = InitSQL(); err != nil {
		log.Fatal("Init SQL error")
	}
	
	if err = InitRedis(); err != nil {
		log.Fatal("Init Redis error")
	}
	if err = InitEs(); err != nil {
		log.Fatal("Init Es error")
	}

}
func sum(s []int, c chan int) {
	sum := 0
	for _, v := range s {
		sum += v
	}
	c <- sum // send sum to c
}

func readWordsFromFile(path string, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	words := make([]string, 0, count)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
		if len(words) >= count {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return words, nil
}

func main() {
	pswd := os.Getenv("RYANQI_PSWD")
	_, err := userLogin("RyanQi", pswd)
	if err != nil {
		log.Fatal("userLogin error:", err)
	}
	err = syncMissingFromLogs()
	if err != nil {
		log.Fatal("syncMissingFromLogs error:", err)
	}
	go checkSyncLog()
	time.Sleep(100 * time.Second)

}