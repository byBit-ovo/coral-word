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
	_ "os"
	_ "strconv"
	_ "time"

	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
	"github.com/joho/godotenv"
	_"strings"
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

func main() {
	words, err, errWords := QueryWords("collaborate")
	if err != nil {
		log.Fatal(err)
	}
	for _, word_desc := range words {
		word_desc.show()
	}
	if len(errWords) > 0 {
		fmt.Println("error words:")
		for _, word := range errWords {
			fmt.Println(word)
		}
	}
}
