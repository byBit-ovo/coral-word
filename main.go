package main

import (
	_ "context"
	"fmt"
	"log"
	"os"
	"github.com/byBit-ovo/coral_word/llm"
	"github.com/joho/godotenv"
	"encoding/json"
)


func main() {
	err := godotenv.Load(".env")
	if err != nil{
		log.Fatal("loading env file err")
	}
	llm.Gemini_api_key = os.Getenv("GEMINI_API_KEY")
	llm.Deepseek_api_key = os.Getenv("DEEPSEEK_API_KEY")
	// client, _ := deepseek.NewClient(Deepseek_api_key)
	model, err := llm.NewAIModel(llm.DEEP_SEEK)
	if err != nil{
		fmt.Print("New Model error: " + err.Error())
	}
	payload, err := model.GetDefinition("compact")
	if err != nil{
		fmt.Print("GetDefinition error: " + err.Error())

	}
	word_1 := wordDesc{}
	fmt.Println(payload)
	err = json.Unmarshal([]byte(payload), &word_1)
	if err != nil{
		fmt.Print("Unmarshal error: " + err.Error())
	}
	showWord(&word_1)
}
