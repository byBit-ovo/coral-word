package main

import (
	"bytes"
	_ "context"
	_ "database/sql"
	"encoding/json"
	_ "encoding/json"
	"fmt"
	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
	"github.com/joho/godotenv"
	"log"
	_ "strconv"
	_ "time"
	"os"
)
func testArticle(words []string){
	article, err := GetArticleDesc(words)
	if err != nil || article.Err == "true"{
		log.Fatal("GetArticle error: " ,err)
	}
	fmt.Println(article.Article)
	fmt.Println(article.Article_cn)
}
func testWord(words []string){
	for _, word := range words{
		res, err := QueryWord(word)
		if err != nil{
			log.Fatal(err)
		}
		res.show()
	}
}
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

}
func sum(s []int, c chan int) {
	sum := 0
	for _, v := range s {
		sum += v
	}
	c <- sum // send sum to c
}

func main() {
	pswd := os.Getenv("RYANQI_PSWD")
	RyanQi, err := userLogin("RyanQi", pswd)
	
	if err != nil {
		log.Fatal("insert user erro:", err)
	}
	words, err := RyanQi.GetSelectedWordNotes("cooperate")
	if err != nil {
		log.Fatal("get selected word notes error:", err)
	}
	for _, word := range words {
		fmt.Println(word.UserName, word.Note)
	}
}


func esSearch() {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}
	res, err := EsClient.Search(
		EsClient.Search.WithIndex("user"), // 索引名称
		EsClient.Search.WithBody(&buf),    // 查询内容
		EsClient.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Fatalf("Error parsing response: %s", err)
	}

	// 打印结果
	fmt.Println(result)
}
