package main

import (
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "fmt"
	"log"
	_ "time"

	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)


func main() {
	err := godotenv.Load(".env")
	if err != nil{
		log.Fatal("Loading env file error")
	}
	if llm.InitModels() != nil{
		log.Fatal("InitModels error")
		return 
	}
	if InitSQL() != nil{
		log.Fatal("Init SQL error")
	}
	word, err := QueryWord("academy")
	if err != nil{
		fmt.Println(err)
	}
	showWord(word)
	// res, err := llm.Models[llm.DEEP_SEEK].GetDefinition("state")
	// word_1 := wordDesc{}
	// err = json.Unmarshal([]byte(res), &word_1)
	// if err != nil{
	// 	fmt.Print("Unmarshal error: " + err.Error())
	// }
	// showWord(&word_1)
	// db, err := sql.Open("mysql", "root:200533@/coral_word")
	// if err != nil {
	// 	panic(err)
	// }
	// defer db.Close()
	// if err := db.Ping(); err != nil {
	// 	log.Fatal("连接失败:", err)
	// }
	// fmt.Println("连接成功！")
	// var word string
	// var pronunciation string
	// var tag int32
	// var example string
	// rows, err := db.Query("select word,pronunciation,tag,example from vocabulary where id=?",1)
	// if err != nil{
	// 	log.Fatal("query err: "+ err.Error())
	// 	return
	// }
	// for rows.Next(){

	// }
	// if err == nil{
	// 	fmt.Println(word,pronunciation,tag,example)
	// }

}

