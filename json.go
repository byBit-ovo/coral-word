package main

import (
	"encoding/json"
	"errors"
	_"fmt"
	"log"

	"github.com/byBit-ovo/coral_word/llm"
	_ "github.com/ydb-platform/ydb-go-sdk/v3/log"
)
import (
	"strings"
)
func processJson(jsonRsp string) string {
	const (
		jsonCodePrefix = "```json"
		codeSuffix     = "```"
		minLength      = 10 
	)
	if len(jsonRsp) < minLength {
		return jsonRsp
	}

	// 修剪首尾空白（处理换行、空格、制表符等），避免标记前后有空白导致匹配失败
	trimmedRsp := strings.TrimSpace(jsonRsp)

	// 检查是否以```json开头、```结尾
	prefixLen := len(jsonCodePrefix)
	suffixLen := len(codeSuffix)
	if strings.HasPrefix(trimmedRsp, jsonCodePrefix) && strings.HasSuffix(trimmedRsp, codeSuffix) {
		content := trimmedRsp[prefixLen : len(trimmedRsp)-suffixLen]
		return strings.TrimSpace(content)
	}
	return jsonRsp
}

func GetWordDesc(word string) (*wordDesc, error){
	choseModel := llm.DEEP_SEEK
	json_rsp, err := llm.Models[choseModel].GetWordDefWithJson(word)
	if err != nil{
		return nil, err
	}
	json_rsp = processJson(json_rsp)
	// fmt.Println(json_rsp)
	word_desc := &wordDesc{}
	word_desc.Source = choseModel
	err = json.Unmarshal([]byte(json_rsp), word_desc)
	if err != nil || word_desc.Err == "true"{
		log.Println("json.Unmarshal error:", err, json_rsp)
		return nil, errors.New("llm returned error response")
	}
	return word_desc, nil
}

func GetArticleDesc(words []string) (*ArticleDesc, error){
	choseModel := llm.DEEP_SEEK
	json_rsp, err := llm.Models[choseModel].GetArticleWithJson(words)
	json_rsp = processJson(json_rsp)
	// fmt.Println(json_rsp)
	if err != nil{
		return nil, err
	}
	article_desc := &ArticleDesc{}
	err = json.Unmarshal([]byte(json_rsp), article_desc)
	if err != nil || article_desc.Err == "true"{
		return nil, errors.New("llm returned error response")
	}
	return article_desc, nil
}
// func resposneWithJson()

