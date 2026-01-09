package main

import (
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	"log"
	_ "time"
	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"encoding/json"
	"bytes"
	_"strconv"
	_"github.com/google/uuid"
	
)
func init(){
	err := godotenv.Load(".env")
	if err != nil{
		log.Fatal("Loading env file error")
	}
	if err = llm.InitModels(); err != nil{
		log.Fatal("InitModels error")
		return 
	}
	if err = InitSQL(); err != nil{
		log.Fatal("Init SQL error")
	}
	if word,err := selectWordById(19);err != nil{
		showWord(word)
	}
	// if err = InitEs(); err != nil{
	// 	log.Fatal("Init es error: ", err)
	// }
	// json_rsp, err := llm.Models[llm.GEMINI].GetDefinition("empathy")
	// fmt.Println(json_rsp)
	// word, err := QueryWord("confront")
	// if err != nil{
	// 	log.Fatal(err)
	// }
	// showWord(word)
	
}

func main() {
	// sid, err := userLogin("byBit","1234567")
	// // user, err := insertUser("byBit","200533")
	// if err != nil{
	// 	log.Fatal("insert user erro:", err)
	// }
	// // if err := createNoteBook(sid, "我的生词本"); err != nil{
	// // 	fmt.Println("create_book error: ", err.Error())
	// // }
	// words := []string{"rely","doom","reveal","debate"}
	// for _,word := range words{
	// 	if err := AddWordToNotebook(sid,word,"我的生词本");err != nil{
	// 		fmt.Println("AddWordNoteBook error: ", err.Error())
	// 	}
	// }
	// fmt.Println(userSession)
	// fmt.Println(wordsPool)
	// fmt.Println(userNoteWords)
	// fmt.Println("sessionId: ", sid)
	// id := uuid.New().String()
	// fmt.Println(id)
	// testGin()
	// word := "set"
	// updateQuery := fmt.Sprintf("update vocabulary set hit_count=hit_count+1 where word = '%s' ", word)
	// fmt.Println(updateQuery)
}

func esSearch(){
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
        EsClient.Search.WithBody(&buf),        // 查询内容
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