package main

import (
	_"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/byBit-ovo/coral_word/llm"
	"github.com/joho/godotenv"
	"database/sql"
	_"time"
	_"github.com/go-sql-driver/mysql"
)


func main() {
	err := godotenv.Load(".env")
	if err != nil{
		log.Fatal("loading env file err")
	}
	llm.InitModels()
	res, err := llm.Models[llm.DEEP_SEEK].GetDefinition("state")
	word_1 := wordDesc{}
	err = json.Unmarshal([]byte(res), &word_1)
	if err != nil{
		fmt.Print("Unmarshal error: " + err.Error())
	}
	showWord(&word_1)
	db, err := sql.Open("mysql", "root:200533@/dbname")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("连接失败:", err)
	}
	fmt.Println("连接成功！")
	
}

