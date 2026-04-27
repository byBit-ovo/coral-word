package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type assistantWordItem struct {
	Word           string   `json:"word"`
	Pronunciation  string   `json:"pronunciation,omitempty"`
	Definitions    []string `json:"definitions,omitempty"`
	EF             float64  `json:"ef"`
	Repetitions    int      `json:"repetitions"`
	IntervalDays   int      `json:"interval_days"`
	NextReviewTime string   `json:"next_review_time,omitempty"`
}

const (
	assistantToolReviewDueWords   = "review_due_words"
	assistantToolReadingRecommend = "reading_recommendation"
	assistantToolFallbackReply    = "fallback_reply"
)

type assistantToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type assistantToolResult struct {
	Intent   string      `json:"intent"`
	Reply    string      `json:"reply"`
	ToolCall string      `json:"tool_call"`
	Data     interface{} `json:"data,omitempty"`
}

type assistantChatRequest struct {
	SessionId string `json:"session_id"`
	BookName  string `json:"book_name"`
	Message   string `json:"message"`
}

func AssistantDueWords(c *gin.Context) {
	var req struct {
		SessionId string `json:"session_id"`
		BookName  string `json:"book_name"`
		Limit     int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SessionId = strings.TrimSpace(c.PostForm("session_id"))
		req.BookName = strings.TrimSpace(c.PostForm("book_name"))
	}
	if req.SessionId == "" {
		req.SessionId = strings.TrimSpace(c.GetHeader("X-Session-ID"))
	}
	if req.SessionId == "" || req.BookName == "" {
		respondError(c, http.StatusBadRequest, "session_id and book_name are required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	items, err := fetchDueWords(uid, req.BookName, req.Limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, gin.H{
		"book_name": req.BookName,
		"count":     len(items),
		"words":     items,
	})
}

func AssistantReadingRecommend(c *gin.Context) {
	var req struct {
		SessionId       string `json:"session_id"`
		BookName        string `json:"book_name"`
		WordLimit       int    `json:"word_limit"`
		GenerateArticle bool   `json:"generate_article"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SessionId = strings.TrimSpace(c.PostForm("session_id"))
		req.BookName = strings.TrimSpace(c.PostForm("book_name"))
	}
	if req.SessionId == "" {
		req.SessionId = strings.TrimSpace(c.GetHeader("X-Session-ID"))
	}
	if req.SessionId == "" || req.BookName == "" {
		respondError(c, http.StatusBadRequest, "session_id and book_name are required")
		return
	}
	if req.WordLimit <= 0 {
		req.WordLimit = 8
	}
	if req.WordLimit > 20 {
		req.WordLimit = 20
	}

	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	recommendWords, details, err := fetchWeakWords(uid, req.BookName, req.WordLimit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	payload := gin.H{
		"book_name":          req.BookName,
		"recommended_words":  recommendWords,
		"recommended_detail": details,
	}

	if req.GenerateArticle {
		article, aerr := GetArticleDescFromLLM(recommendWords)
		if aerr != nil {
			respondError(c, http.StatusInternalServerError, "generate article failed: "+aerr.Error())
			return
		}
		payload["article"] = article
	}
	respondOK(c, payload)
}

func AssistantChat(c *gin.Context) {
	req := parseAssistantChatRequest(c)
	if req.SessionId == "" || req.BookName == "" || strings.TrimSpace(req.Message) == "" {
		respondError(c, http.StatusBadRequest, "session_id, book_name and message are required")
		return
	}

	toolCall := resolveAssistantToolCall(req.Message)
	result, err := executeAssistantToolCall(req, toolCall)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondOK(c, gin.H{
		"intent":    result.Intent,
		"reply":     result.Reply,
		"tool_call": gin.H{"name": toolCall.Name, "arguments": toolCall.Arguments},
		"data":      result.Data,
	})
}

func parseAssistantChatRequest(c *gin.Context) assistantChatRequest {
	var req assistantChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SessionId = strings.TrimSpace(c.PostForm("session_id"))
		req.BookName = strings.TrimSpace(c.PostForm("book_name"))
		req.Message = strings.TrimSpace(c.PostForm("message"))
	}
	if req.SessionId == "" {
		req.SessionId = strings.TrimSpace(c.GetHeader("X-Session-ID"))
	}
	return req
}

func resolveAssistantToolCall(message string) assistantToolCall {
	msg := strings.ToLower(strings.TrimSpace(message))

	if strings.Contains(msg, "复习") || strings.Contains(msg, "review") || strings.Contains(msg, "due") {
		return assistantToolCall{
			Name: assistantToolReviewDueWords,
			Arguments: map[string]interface{}{
				"limit": 10,
			},
		}
	}

	if strings.Contains(msg, "阅读") || strings.Contains(msg, "article") || strings.Contains(msg, "recommend") {
		return assistantToolCall{
			Name: assistantToolReadingRecommend,
			Arguments: map[string]interface{}{
				"word_limit":       8,
				"generate_article": true,
			},
		}
	}

	return assistantToolCall{
		Name:      assistantToolFallbackReply,
		Arguments: map[string]interface{}{},
	}
}

func executeAssistantToolCall(req assistantChatRequest, toolCall assistantToolCall) (*assistantToolResult, error) {
	switch toolCall.Name {
	case assistantToolReviewDueWords:
		return executeReviewDueWordsTool(req, toolCall)
	case assistantToolReadingRecommend:
		return executeReadingRecommendTool(req, toolCall)
	case assistantToolFallbackReply:
		return &assistantToolResult{
			Intent:   "unknown",
			Reply:    "我可以帮你做两件事：1) 单词复习 2) 阅读推荐。你可以说“今天复习什么”或“给我一篇推荐阅读”。",
			ToolCall: toolCall.Name,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported tool call: %s", toolCall.Name)
	}
}

func executeReviewDueWordsTool(req assistantChatRequest, toolCall assistantToolCall) (*assistantToolResult, error) {
	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		return nil, err
	}
	limit := intArg(toolCall.Arguments, "limit", 10)
	items, err := fetchDueWords(uid, req.BookName, limit)
	if err != nil {
		return nil, err
	}

	return &assistantToolResult{
		Intent:   assistantToolReviewDueWords,
		Reply:    "这些是你当前优先复习的单词，建议从 EF 低、到期更早的开始。",
		ToolCall: toolCall.Name,
		Data:     items,
	}, nil
}

func executeReadingRecommendTool(req assistantChatRequest, toolCall assistantToolCall) (*assistantToolResult, error) {
	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		return nil, err
	}
	wordLimit := intArg(toolCall.Arguments, "word_limit", 8)
	generateArticle := boolArg(toolCall.Arguments, "generate_article", true)

	words, details, err := fetchWeakWords(uid, req.BookName, wordLimit)
	if err != nil {
		return nil, err
	}

	data := gin.H{
		"recommended_words":  words,
		"recommended_detail": details,
	}
	if generateArticle {
		article, err := GetArticleDescFromLLM(words)
		if err != nil {
			return nil, fmt.Errorf("generate article failed: %w", err)
		}
		data["article"] = article
	}

	return &assistantToolResult{
		Intent:   assistantToolReadingRecommend,
		Reply:    "我根据你当前薄弱词推荐了一篇阅读材料，先读英文，再对照中文。",
		ToolCall: toolCall.Name,
		Data:     data,
	}, nil
}

func intArg(args map[string]interface{}, key string, fallback int) int {
	raw, ok := args[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func boolArg(args map[string]interface{}, key string, fallback bool) bool {
	raw, ok := args[key]
	if !ok {
		return fallback
	}
	v, ok := raw.(bool)
	if !ok {
		return fallback
	}
	return v
}

func fetchDueWords(uid, bookName string, limit int) ([]assistantWordItem, error) {
	rows, err := db.Query(`SELECT 
		v.word, v.pronunciation,
		lr.ef, lr.repetitions, lr.interval_days, lr.next_review_time
		FROM learning_record lr
		JOIN vocabulary v ON lr.word_id = v.id
		WHERE lr.user_id = ? AND lr.book_name = ?
		AND (lr.next_review_time <= NOW() OR lr.next_review_time IS NULL)
		ORDER BY lr.ef ASC, lr.next_review_time ASC
		LIMIT ?`, uid, bookName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []assistantWordItem
	for rows.Next() {
		var item assistantWordItem
		var nextReview sql.NullTime
		var interval sql.NullInt64
		var ef sql.NullFloat64
		var reps sql.NullInt64
		if err := rows.Scan(&item.Word, &item.Pronunciation, &ef, &reps, &interval, &nextReview); err != nil {
			return nil, err
		}
		if ef.Valid {
			item.EF = ef.Float64
		}
		if reps.Valid {
			item.Repetitions = int(reps.Int64)
		}
		if interval.Valid {
			item.IntervalDays = int(interval.Int64)
		}
		if nextReview.Valid {
			item.NextReviewTime = nextReview.Time.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fillWordDefinitions(items), nil
}

func fetchWeakWords(uid, bookName string, limit int) ([]string, []assistantWordItem, error) {
	rows, err := db.Query(`SELECT 
		v.word, v.pronunciation,
		lr.ef, lr.repetitions, lr.interval_days, lr.next_review_time
		FROM learning_record lr
		JOIN vocabulary v ON lr.word_id = v.id
		WHERE lr.user_id = ? AND lr.book_name = ?
		ORDER BY lr.ef ASC, lr.repetitions ASC, lr.next_review_time ASC
		LIMIT ?`, uid, bookName, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var items []assistantWordItem
	for rows.Next() {
		var item assistantWordItem
		var nextReview sql.NullTime
		var interval sql.NullInt64
		var ef sql.NullFloat64
		var reps sql.NullInt64
		if err := rows.Scan(&item.Word, &item.Pronunciation, &ef, &reps, &interval, &nextReview); err != nil {
			return nil, nil, err
		}
		if ef.Valid {
			item.EF = ef.Float64
		}
		if reps.Valid {
			item.Repetitions = int(reps.Int64)
		}
		if interval.Valid {
			item.IntervalDays = int(interval.Int64)
		}
		if nextReview.Valid {
			item.NextReviewTime = nextReview.Time.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	items = fillWordDefinitions(items)
	sort.Slice(items, func(i, j int) bool {
		if items[i].EF == items[j].EF {
			return items[i].Word < items[j].Word
		}
		return items[i].EF < items[j].EF
	})
	words := make([]string, 0, len(items))
	for _, item := range items {
		words = append(words, item.Word)
	}
	return words, items, nil
}

func fillWordDefinitions(items []assistantWordItem) []assistantWordItem {
	if len(items) == 0 {
		return items
	}
	words := make([]string, 0, len(items))
	for _, item := range items {
		words = append(words, item.Word)
	}
	descs, err, _ := QueryWords(words...)
	if err != nil {
		return items
	}

	for i := range items {
		wd, ok := descs[items[i].Word]
		if !ok || wd == nil {
			continue
		}
		defs := make([]string, 0, len(wd.Definitions))
		for _, def := range wd.Definitions {
			if len(def.Meanings) == 0 {
				continue
			}
			defs = append(defs, strings.Join(def.Meanings, "；"))
		}
		items[i].Definitions = defs
	}
	return items
}
