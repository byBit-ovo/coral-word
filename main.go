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
	_"time"
	
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
	// var word *wordDesc
	// if word,err = selectWordById(18);err != nil{
	// 	fmt.Println(err)
	// }
	// showWord(word)
	// if err = InitEs(); err != nil{
	// 	log.Fatal("Init es error: ", err)
	// }
	// json_rsp, err := llm.Models[llm.GEMINI].GetDefinition("empathy")
	// fmt.Println(json_rsp)
	word, err := QueryWord("kidnap")
	if err != nil{
		log.Fatal(err)
	}
	showWord(word)
	
}

func main() {
	_, err := userLogin("byBit","1234567")
	// user, err := insertUser("byBit","200533")
	if err != nil{
		log.Fatal("insert user erro:", err)
	}
	// now := time.Now()
    
    // fmt.Println("当前时间:", now)
    // fmt.Println("年:", now.Year())
    // fmt.Println("月:", now.Month())
    // fmt.Println("日:", now.Day())
    // fmt.Println("时:", now.Hour())
    // fmt.Println("分:", now.Minute())
    // fmt.Println("秒:", now.Second())
    // fmt.Println("纳秒:", now.Nanosecond())
    // fmt.Println("星期:", now.Weekday())
    // fmt.Println("Unix时间戳:", now.Unix())
    // fmt.Println("Unix毫秒:", now.UnixMilli())
	// fmt.Println(now.Format("2006-01-02"))           // 2024-01-15
	// fmt.Println(now.Format("2006-01-02 15:04:05"))  // 2024-01-15 14:30:45
	// fmt.Println(now.Format("2006/01/02"))           // 2024/01/15
	// fmt.Println(now.Format("15:04:05"))             // 14:30:45
	// fmt.Println(now.Format("2006年01月02日"))        // 2024年01月15日
	// fmt.Println(now.Format(time.RFC3339))           // 2024-01-15T14:30:45+08:00

	// 12小时制
	// fmt.Println(now.Format("03:04:05 PM"))
	// // if err := createNoteBook(sid, "我的生词本"); err != nil{
	// // 	fmt.Println("create_book error: ", err.Error())
	// // }
	// words := []string{"rely","doom","reveal","debate","metabolism","cyber","stock"}
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