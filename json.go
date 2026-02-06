package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "github.com/byBit-ovo/coral_word/pb"
	"github.com/byBit-ovo/coral_word/LLM"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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



// 批量查询：若设置 LLM_GRPC_TARGET 则走 gRPC；若设置 LLM_SERVICE_URL 则走 HTTP；否则直连 Go LLM
func GetWordDescFromLLM(words ...string) (map[string]*wordDesc, error) {
	if target := strings.TrimSpace(os.Getenv("LLM_GRPC_TARGET")); target != "" {
		return getWordDescFromPythonGrpc(target, words...)
	}
	if baseURL := strings.TrimSuffix(os.Getenv("LLM_SERVICE_URL"), "/"); baseURL != "" {
		return getWordDescFromPythonService(baseURL, words...)
	}
	choseModel := llm.DEEP_SEEK
	jsonRsp, err := llm.Models[choseModel].GetWordDefWithJson(words...)
	if err != nil {
		return nil, err
	}
	jsonRsp = processJson(jsonRsp)
	var wrapper struct {
		Words []*wordDesc `json:"words"`
	}
	wrapper.Words = make([]*wordDesc, len(words))
	for i := range wrapper.Words {
		wrapper.Words[i] = &wordDesc{}
		wrapper.Words[i].LLMModelName = llm.ModelsName[choseModel]
		wrapper.Words[i].Word = words[i]
	}
	if err := json.Unmarshal([]byte(jsonRsp), &wrapper); err != nil {
		log.Println("json.Unmarshal error:", err, jsonRsp)
		return nil, errors.New("llm returned error response")
	}
	res := make(map[string]*wordDesc)
	for _, wd := range wrapper.Words {
		res[wd.Word] = wd
	}
	return res, nil
}

func getWordDescFromPythonGrpc(target string, words ...string) (map[string]*wordDesc, error) {
	if !strings.Contains(target, "://") {
		target = "passthrough:///" + target
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("llm grpc dial: %w", err)
	}
	defer conn.Close()
	client := pb.NewLLMServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resp, err := client.WordDefinitions(ctx, &pb.WordDefinitionsRequest{Words: words})
	if err != nil {
		return nil, fmt.Errorf("llm grpc WordDefinitions: %w", err)
	}
	res := make(map[string]*wordDesc)
	for _, p := range resp.GetWords() {
		wd := FromPbWordDesc(p)
		if wd != nil {
			res[wd.Word] = wd
		}
	}
	return res, nil
}

func getWordDescFromPythonService(baseURL string, words ...string) (map[string]*wordDesc, error) {
	body, _ := json.Marshal(map[string][]string{"words": words})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/word_definitions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm service request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm service status %d", resp.StatusCode)
	}
	var wrapper struct {
		Words []*wordDesc `json:"words"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("llm service decode: %w", err)
	}
	res := make(map[string]*wordDesc)
	for _, wd := range wrapper.Words {
		res[wd.Word] = wd
	}
	return res, nil
}

func GetArticleDescFromLLM(words []string) (*ArticleDesc, error) {
	if target := strings.TrimSpace(os.Getenv("LLM_GRPC_TARGET")); target != "" {
		return getArticleFromPythonGrpc(target, words)
	}
	if baseURL := strings.TrimSuffix(os.Getenv("LLM_SERVICE_URL"), "/"); baseURL != "" {
		return getArticleFromPythonService(baseURL, words)
	}
	choseModel := llm.DEEP_SEEK
	jsonRsp, err := llm.Models[choseModel].GetArticleWithJson(words)
	if err != nil {
		return nil, err
	}
	jsonRsp = processJson(jsonRsp)
	articleDesc := &ArticleDesc{}
	if err := json.Unmarshal([]byte(jsonRsp), articleDesc); err != nil || articleDesc.Err == "true" {
		return nil, errors.New("llm returned error response")
	}
	return articleDesc, nil
}

func getArticleFromPythonGrpc(target string, words []string) (*ArticleDesc, error) {
	if !strings.Contains(target, "://") {
		target = "passthrough:///" + target
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("llm grpc dial: %w", err)
	}
	defer conn.Close()
	client := pb.NewLLMServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resp, err := client.Article(ctx, &pb.ArticleRequest{Words: words})
	if err != nil {
		return nil, fmt.Errorf("llm grpc Article: %w", err)
	}
	if resp.GetError() == "true" {
		return nil, errors.New("llm returned error response")
	}
	return &ArticleDesc{
		Err:        resp.GetError(),
		Article:    resp.GetArticle(),
		Article_cn: resp.GetArticleCn(),
	}, nil
}

func getArticleFromPythonService(baseURL string, words []string) (*ArticleDesc, error) {
	body, _ := json.Marshal(map[string][]string{"words": words})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/article", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm service request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm service status %d", resp.StatusCode)
	}
	var articleDesc ArticleDesc
	if err := json.NewDecoder(resp.Body).Decode(&articleDesc); err != nil {
		return nil, fmt.Errorf("llm service decode: %w", err)
	}
	if articleDesc.Err == "true" {
		return nil, errors.New("llm returned error response")
	}
	return &articleDesc, nil
}
// func resposneWithJson()

