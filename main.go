package main

import (
	_ "bufio"
	_ "bytes"
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	"log"
	_ "net/http"
	"os"
	_ "strconv"
	"time"
	_ "time"

	"bufio"
	"github.com/byBit-ovo/coral_word/LLM"
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)

}

func f1() {
	x := 10
	defer fmt.Println(x)
	x = 20
}
func f2() {
	x := 10
	defer func() {
		fmt.Println(x)
	}()
	x = 20
}
func test() {

	words := []string{"revoke", "impose", "virtually", "profound"}
	word_descs, err, errWords := QueryWords(words...)
	if err != nil {
		log.Println("QueryWords error:", err, "errWords:", errWords)
		return
	}
	for _, word_desc := range word_descs {
		word_desc.show()
	}
	time.Sleep(10 * time.Hour)
}
func offLineMode(){
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("珊瑚英语单词查询系统,输入单词查询:")
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			fmt.Println("请输入单词查询:")
			continue
		}
		word_descs, err, errWords := QueryWords(word)
		if err != nil {
			log.Println("QueryWords error:", err, "errWords:", errWords)
			continue
		}
		for _, word_desc := range word_descs {
			word_desc.show()
		}
	}
}
// LLMPool 全局协程池，用于查询 LLM 补全单词（用户查词时复用，避免每次起新 goroutine）
var LLMPool *GoRoutinePool

func main() {
	LLMPool = NewPool(10, 200)
	defer LLMPool.Shutdown()
	RunHTTPServer(os.Getenv("HTTP_ADDR"))
}
