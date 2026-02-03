package main

import (
	"encoding/json"
	"errors"
	_"fmt"
	"log"
	"strings"

	"github.com/byBit-ovo/coral_word/LLM"
	_ "github.com/ydb-platform/ydb-go-sdk/v3/log"
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



// 批量查询
func GetWordDescFromLLM(words ...string) (map[string]*wordDesc, error){
	choseModel := llm.DEEP_SEEK
	json_rsp, err := llm.Models[choseModel].GetWordDefWithJson(words...)
	if err != nil{
		return nil, err
	}
	json_rsp = processJson(json_rsp)
	// fmt.Println(json_rsp)
	var wrapper struct {
		Words []*wordDesc `json:"words"`
	}
	wrapper.Words = make([]*wordDesc, len(words))
	for i, _ := range wrapper.Words{
		wrapper.Words[i] = &wordDesc{}
		wrapper.Words[i].LLMModelName = llm.ModelsName[choseModel]
		wrapper.Words[i].Word = words[i]
	}
	err = json.Unmarshal([]byte(json_rsp), &wrapper)
	if err != nil {
		log.Println("json.Unmarshal error:", err, json_rsp)
		return nil, errors.New("llm returned error response")
	}
	res := make(map[string]*wordDesc)
	for _, word_desc := range wrapper.Words {
		res[word_desc.Word] = word_desc
	}
	return res, nil
}

func GetArticleDescFromLLM(words []string) (*ArticleDesc, error){
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

