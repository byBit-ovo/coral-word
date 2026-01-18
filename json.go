package main

import (
	"encoding/json"
	_"fmt"

	"github.com/byBit-ovo/coral_word/llm"
	_"github.com/ydb-platform/ydb-go-sdk/v3/log"
)


func GetWordDesc(word string) (*wordDesc, error){
	choseModel := llm.DEEP_SEEK
	json_rsp, err := llm.Models[choseModel].GetWordDefWithJson(word)
	if err != nil{
		return nil, err
	}
	word_desc := &wordDesc{}
	word_desc.Source = choseModel
	err = json.Unmarshal([]byte(json_rsp), word_desc)
	if err != nil || word_desc.Err == "true"{
		return nil, err
	}
	return word_desc, nil
}

func GetArticleDesc(words []string) (*ArticleDesc, error){
	choseModel := llm.DEEP_SEEK
	json_rsp, err := llm.Models[choseModel].GetArticleWithJson(words)
	if err != nil{
		return nil, err
	}
	article_desc := &ArticleDesc{}
	err = json.Unmarshal([]byte(json_rsp), article_desc)
	if err != nil || article_desc.Err == "true"{
		return nil, err
	}
	return article_desc, nil
}
// func resposneWithJson()

