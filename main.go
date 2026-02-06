package main

import (
	_ "bufio"
	_ "bytes"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	"log"
	_ "net/http"
	"os"
	_"strconv"
	_ "strconv"
	"time"
	_ "time"
	"bufio"

	"github.com/byBit-ovo/coral_word/LLM"
	"github.com/joho/godotenv"
	_"golang.org/x/sync/singleflight"
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
	LLMPool = NewPool(10, 200)

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

func testSf(num *int64)error{
	*num = *num + 1
	return nil
}

const(
	s1 = "hello"
)

func main() {
	//"64a3a609-85d3-44ff-8f41-4efcd7a4a975"
	defer LLMPool.Shutdown()
	RunHTTPServer(os.Getenv("HTTP_ADDR"))
	// go RunGrpcServer(os.Getenv("GRPC_ADDR"))
	// grcpClient, err := NewCoralWordGrpcClient()
	// if err != nil {
	// 	log.Fatalf("failed to create grpc client: %v", err)
	// }
	// for true{
	// 	var word string
	// 	fmt.Scan(&word)
	// 	word_descs, err := grcpClient.QueryWord(context.Background(), word)
	// 	if err != nil {
	// 		log.Fatalf("failed to query word: %v", err)
	// 	}
	// 	if word_descs.Err != "false" {
	// 		log.Println(word_descs.Message)
	// 		continue
	// 	} 
	// 	for _, word_desc := range word_descs.GetWordDescs() {
	// 		FromPbWordDesc(word_desc).show()
	// 	}

	// }
	// StartReview("ab7b3f22-861f-4288-b8f5-46676bb0042a","我的生词本")
	// RyanQi := &User{
	// 	Name: "RyanQi",	
	// 	Pswd: "1234567",
	// }
	// err := RyanQi.userLogin()
	// if err != nil {
	// 	log.Println("userLogin error:", err)
	// 	return
	// }
	// log.Println("userLogin success")
	// time.Sleep(10 * time.Second)
	// err = RyanQi.userLogout()
	// if err != nil {
	// 	log.Println("userLogout error:", err)
	// 	return
	// }
	// log.Println("userLogout success")

}
