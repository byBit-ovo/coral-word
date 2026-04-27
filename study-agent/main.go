package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const timeFormat = time.RFC3339

type Word struct {
	Term            string    `json:"term"`
	Meaning         string    `json:"meaning"`
	Example         string    `json:"example"`
	Tags            []string  `json:"tags,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DueAt           time.Time `json:"due_at"`
	IntervalDays    int       `json:"interval_days"`
	EaseFactor      float64   `json:"ease_factor"`
	RepetitionCount int       `json:"repetition_count"`
	LastScore       int       `json:"last_score"`
}

type ReviewRecord struct {
	Term       string    `json:"term"`
	Score      int       `json:"score"`
	ReviewedAt time.Time `json:"reviewed_at"`
	NextDueAt  time.Time `json:"next_due_at"`
}

type ReadingItem struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Difficulty int      `json:"difficulty"`
	Topics     []string `json:"topics"`
	Reason     string   `json:"reason,omitempty"`
}

type MemoryStore struct {
	Words         map[string]*Word `json:"words"`
	ReviewHistory []ReviewRecord   `json:"review_history"`
}

type App struct {
	mu           sync.Mutex
	store        MemoryStore
	dataPath     string
	readingItems []ReadingItem
}

type addWordRequest struct {
	Term    string   `json:"term"`
	Meaning string   `json:"meaning"`
	Example string   `json:"example"`
	Tags    []string `json:"tags"`
}

type reviewRequest struct {
	Term  string `json:"term"`
	Score int    `json:"score"`
}

type recommendRequest struct {
	Topics       []string `json:"topics"`
	CurrentLevel int      `json:"current_level"`
	Limit        int      `json:"limit"`
}

type assistantRequest struct {
	Message string `json:"message"`
}

type assistantResponse struct {
	Intent string      `json:"intent"`
	Reply  string      `json:"reply"`
	Data   interface{} `json:"data,omitempty"`
}

func main() {
	dataPath := cmp.Or(os.Getenv("STUDY_AGENT_DATA"), "./data/study_data.json")

	app := &App{
		dataPath: dataPath,
		store: MemoryStore{
			Words:         map[string]*Word{},
			ReviewHistory: []ReviewRecord{},
		},
		readingItems: defaultReadings(),
	}

	if err := app.load(); err != nil {
		log.Fatalf("load store: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.healthHandler)
	mux.HandleFunc("POST /words", app.addWordHandler)
	mux.HandleFunc("GET /words/due", app.dueWordsHandler)
	mux.HandleFunc("POST /review", app.reviewWordHandler)
	mux.HandleFunc("POST /reading/recommend", app.readingRecommendHandler)
	mux.HandleFunc("POST /assistant/chat", app.assistantChatHandler)

	port := cmp.Or(os.Getenv("PORT"), "9030")
	addr := "localhost:" + port
	log.Printf("study-agent listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, loggingMiddleware(mux)))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func (a *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	renderJSON(w, map[string]string{"status": "ok", "time": time.Now().Format(timeFormat)})
}

func (a *App) addWordHandler(w http.ResponseWriter, req *http.Request) {
	var in addWordRequest
	if err := readJSON(req, &in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in.Term = strings.TrimSpace(strings.ToLower(in.Term))
	if in.Term == "" || strings.TrimSpace(in.Meaning) == "" {
		http.Error(w, "term and meaning are required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()

	word, ok := a.store.Words[in.Term]
	if !ok {
		word = &Word{
			Term:            in.Term,
			CreatedAt:       now,
			IntervalDays:    1,
			EaseFactor:      2.5,
			RepetitionCount: 0,
			DueAt:           now,
		}
		a.store.Words[in.Term] = word
	}

	word.Meaning = strings.TrimSpace(in.Meaning)
	word.Example = strings.TrimSpace(in.Example)
	word.Tags = uniqueClean(in.Tags)
	word.UpdatedAt = now

	if err := a.saveLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, word)
}

func (a *App) dueWordsHandler(w http.ResponseWriter, req *http.Request) {
	limit := 10
	if raw := strings.TrimSpace(req.URL.Query().Get("limit")); raw != "" {
		if n, err := parsePositiveInt(raw); err == nil {
			limit = min(n, 100)
		}
	}

	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()

	due := make([]Word, 0)
	for _, wd := range a.store.Words {
		if !wd.DueAt.After(now) {
			due = append(due, *wd)
		}
	}

	sort.Slice(due, func(i, j int) bool { return due[i].DueAt.Before(due[j].DueAt) })
	if len(due) > limit {
		due = due[:limit]
	}

	renderJSON(w, map[string]interface{}{
		"count": len(due),
		"words": due,
	})
}

func (a *App) reviewWordHandler(w http.ResponseWriter, req *http.Request) {
	var in reviewRequest
	if err := readJSON(req, &in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	term := strings.TrimSpace(strings.ToLower(in.Term))
	if term == "" {
		http.Error(w, "term is required", http.StatusBadRequest)
		return
	}
	if in.Score < 0 || in.Score > 5 {
		http.Error(w, "score must be between 0 and 5", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	word, ok := a.store.Words[term]
	if !ok {
		http.Error(w, "word not found", http.StatusNotFound)
		return
	}

	now := time.Now()
	nextDue, intervalDays, easeFactor, reps := applySM2(word.IntervalDays, word.EaseFactor, word.RepetitionCount, in.Score, now)

	word.IntervalDays = intervalDays
	word.EaseFactor = easeFactor
	word.RepetitionCount = reps
	word.LastScore = in.Score
	word.DueAt = nextDue
	word.UpdatedAt = now

	a.store.ReviewHistory = append(a.store.ReviewHistory, ReviewRecord{
		Term:       term,
		Score:      in.Score,
		ReviewedAt: now,
		NextDueAt:  nextDue,
	})

	if err := a.saveLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, map[string]interface{}{
		"word":        word,
		"next_due_at": nextDue.Format(timeFormat),
	})
}

func (a *App) readingRecommendHandler(w http.ResponseWriter, req *http.Request) {
	var in recommendRequest
	if err := readJSON(req, &in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}

	topicSet := toSet(uniqueClean(in.Topics))
	if len(topicSet) == 0 {
		topicSet["go-basics"] = struct{}{}
	}
	if in.CurrentLevel <= 0 {
		in.CurrentLevel = 2
	}

	type scored struct {
		item  ReadingItem
		score int
	}

	out := make([]scored, 0, len(a.readingItems))
	for _, item := range a.readingItems {
		topicMatch := countTopicMatches(topicSet, item.Topics)
		difficultyGap := abs(item.Difficulty - in.CurrentLevel)
		score := topicMatch*10 - difficultyGap*3
		reason := buildReason(topicMatch, difficultyGap, item.Topics)
		item.Reason = reason
		out = append(out, scored{item: item, score: score})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].item.Difficulty < out[j].item.Difficulty
		}
		return out[i].score > out[j].score
	})

	res := make([]ReadingItem, 0, limit)
	for i := 0; i < len(out) && i < limit; i++ {
		res = append(res, out[i].item)
	}

	renderJSON(w, map[string]interface{}{
		"recommendations": res,
	})
}

func (a *App) assistantChatHandler(w http.ResponseWriter, req *http.Request) {
	var in assistantRequest
	if err := readJSON(req, &in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := strings.TrimSpace(strings.ToLower(in.Message))
	if msg == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	if strings.Contains(msg, "复习") || strings.Contains(msg, "review") || strings.Contains(msg, "due") {
		a.mu.Lock()
		now := time.Now()
		due := make([]Word, 0)
		for _, wd := range a.store.Words {
			if !wd.DueAt.After(now) {
				due = append(due, *wd)
			}
		}
		a.mu.Unlock()

		sort.Slice(due, func(i, j int) bool { return due[i].DueAt.Before(due[j].DueAt) })
		if len(due) > 5 {
			due = due[:5]
		}

		renderJSON(w, assistantResponse{
			Intent: "word_review",
			Reply:  "已为你准备待复习单词，建议先从最早到期的开始。",
			Data:   due,
		})
		return
	}

	if strings.Contains(msg, "阅读") || strings.Contains(msg, "read") || strings.Contains(msg, "recommend") {
		fakeReq := recommendRequest{Topics: []string{"go-basics"}, CurrentLevel: 2, Limit: 3}
		buf, _ := json.Marshal(fakeReq)
		r := req.Clone(req.Context())
		r.Body = io.NopCloser(bytes.NewReader(buf))
		a.readingRecommendHandler(w, r)
		return
	}

	renderJSON(w, assistantResponse{
		Intent: "smalltalk",
		Reply:  "我可以帮你做两件事：1) 单词复习 2) 阅读推荐。你可以说：'今天要复习什么单词？' 或 '给我推荐阅读'。",
	})
}

func (a *App) load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(a.dataPath), 0o755); err != nil {
		return err
	}

	b, err := os.ReadFile(a.dataPath)
	if errors.Is(err, os.ErrNotExist) {
		return a.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return a.saveLocked()
	}

	var in MemoryStore
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	if in.Words == nil {
		in.Words = map[string]*Word{}
	}
	a.store = in
	return nil
}

func (a *App) saveLocked() error {
	b, err := json.MarshalIndent(a.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.dataPath, b, 0o644)
}

func applySM2(intervalDays int, easeFactor float64, repetitions int, quality int, now time.Time) (time.Time, int, float64, int) {
	ef := easeFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if ef < 1.3 {
		ef = 1.3
	}

	if quality < 3 {
		repetitions = 0
		intervalDays = 1
	} else {
		repetitions++
		switch repetitions {
		case 1:
			intervalDays = 1
		case 2:
			intervalDays = 6
		default:
			intervalDays = int(math.Round(float64(max(intervalDays, 1)) * ef))
		}
	}

	next := now.Add(time.Duration(intervalDays) * 24 * time.Hour)
	return next, intervalDays, ef, repetitions
}

func defaultReadings() []ReadingItem {
	return []ReadingItem{
		{
			ID:         "go-basics-hello",
			Title:      "Hello + Reverse: Go 基础与测试",
			Path:       "example/hello/hello.go",
			Difficulty: 1,
			Topics:     []string{"go-basics", "testing"},
		},
		{
			ID:         "go-web-gin",
			Title:      "Web Service Gin 实战",
			Path:       "web-service-gin/main.go",
			Difficulty: 2,
			Topics:     []string{"http", "gin", "go-basics"},
		},
		{
			ID:         "go-generics",
			Title:      "Go 泛型入门",
			Path:       "generics/main.go",
			Difficulty: 2,
			Topics:     []string{"generics", "go-basics"},
		},
		{
			ID:         "go-types-guide",
			Title:      "go/types 指南",
			Path:       "example/gotypes/README.md",
			Difficulty: 4,
			Topics:     []string{"analysis", "compiler", "advanced"},
		},
		{
			ID:         "ragserver-core",
			Title:      "RAG Server 主流程",
			Path:       "example/ragserver/ragserver/main.go",
			Difficulty: 3,
			Topics:     []string{"rag", "http", "llm"},
		},
	}
}

func readJSON(req *http.Request, out interface{}) error {
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func renderJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func uniqueClean(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		x := strings.TrimSpace(strings.ToLower(v))
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func toSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, x := range items {
		set[x] = struct{}{}
	}
	return set
}

func countTopicMatches(topicSet map[string]struct{}, topics []string) int {
	count := 0
	for _, topic := range topics {
		if _, ok := topicSet[topic]; ok {
			count++
		}
	}
	return count
}

func buildReason(topicMatch int, gap int, topics []string) string {
	if topicMatch > 0 && gap <= 1 {
		return "主题匹配且难度接近当前水平"
	}
	if topicMatch > 0 {
		return "主题匹配，适合作为进阶阅读"
	}
	if gap <= 1 {
		return "主题泛化但难度合适，可用于拓展"
	}
	return "用于扩展阅读面"
}

func parsePositiveInt(raw string) (int, error) {
	var n int
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, errors.New("must be positive")
	}
	return n, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
