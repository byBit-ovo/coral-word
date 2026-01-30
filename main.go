package main

import (
	_ "bytes"
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"log"
	_ "strconv"
	"time"
	_"net/http"
	"fmt"
	"github.com/byBit-ovo/coral_word/llm"
	_"github.com/gin-gonic/gin"
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)

}


func main() {
	// scaleUpWords(100)
	// syncMissingFromLogs()
	// checkSyncLog()
	RyanQi := User{
		Name: "RyanQi",
		Pswd: "1234567",
	}
	err := RyanQi.userLogin()
	if err != nil{
		log.Fatal(err)
	}
	
	fmt.Println(RyanQi.SessionId)
	user_id,err := redisClient.GetUserSession(RyanQi.SessionId)
	if err != nil{
		log.Fatal(err)
	}
	fmt.Println("user_id:",user_id)
	note,err :=RyanQi.GetWordNote("reveal")
	if err != nil{
		log.Fatal(err)
	}
	fmt.Println(note.Note)
	time.Sleep(10 * time.Second)
	RyanQi.userLogout()


}