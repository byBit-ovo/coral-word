package main

import (
	_ "errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //如果字段是“空值”，就不要出现在 JSON 里。
}

var (
	reviewSessions   = map[string]*ReviewSession{}
	reviewSessionsMu sync.Mutex
)

func RunHTTPServer(addr string) error {
	router := gin.Default()

	router.POST("/login", Login)
	router.POST("/register", Register)
	router.GET("/word", WordQuery)
	router.POST("/create_note", CreateNote)
	router.PUT("/update_note", UpdateNote)
	router.GET("/get_note", GetNote)
	router.DELETE("/delete_note", DeleteNote)
	router.POST("/create_note_book", CreateNoteBookApi)
	router.POST("/add_word_to_notebook", AddWordToNotebookApi)
	router.POST("/review/start", ReviewStart)
	router.GET("/review/next", NextReview)
	router.POST("/review/submit", SubmitReview)

	return router.Run(addr)
}

func Login(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
		Pswd string `json:"pswd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = c.PostForm("name")
		req.Pswd = c.PostForm("pswd")
	}
	if req.Name == "" || req.Pswd == "" {
		respondError(c, http.StatusBadRequest, "name or password is empty")
		return
	}
	user := &User{
		Name: req.Name,
		Pswd: req.Pswd,
	}
	fmt.Println(user)
	if err := user.userLogin(); err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	respondOK(c, gin.H{"user": user})
}

func Register(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
		Pswd string `json:"pswd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = c.PostForm("name")
		req.Pswd = c.PostForm("pswd")
	}
	if req.Name == "" || req.Pswd == "" {
		respondError(c, http.StatusBadRequest, "name or password is empty")
		return
	}
	user := &User{
		Name: req.Name,
		Pswd: req.Pswd,
	}
	if err := user.userRegister(); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondOK(c, gin.H{"user": user})
}

func WordQuery(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		respondError(c, http.StatusBadRequest, "word is empty")
		return
	}
	word_desc, err, missWords := QueryWords(word)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, gin.H{"words": word_desc, "miss_words": missWords})
}

func CreateNote(c *gin.Context) {

	var req struct {
		UserId    string `json:"user_id"`
		SessionId string `json:"session_id"`
		Word      string `json:"word"`
		Note      string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Word = c.PostForm("word")
		req.Note = c.PostForm("note")
		req.UserId = c.PostForm("user_id")
		req.SessionId = c.PostForm("session_id")
	}
	if req.Word == "" {
		respondError(c, http.StatusBadRequest, "word is empty")
		return
	}
	name, err := redisClient.GetUserName(req.UserId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	user := &User{
		Id:        req.UserId,
		Name:      name,
		SessionId: req.SessionId,
	}
	if err := user.CreateWordNote(req.Word, req.Note); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, nil)
}

func UpdateNote(c *gin.Context) {
	var req struct {
		UserId    string `json:"user_id"`
		SessionId string `json:"session_id"`
		Word      string `json:"word"`
		Note      string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Word = c.PostForm("word")
		req.Note = c.PostForm("note")
		req.UserId = c.PostForm("user_id")
		req.SessionId = c.PostForm("session_id")
	}
	if req.Word == "" {
		respondError(c, http.StatusBadRequest, "word is empty")
		return
	}
	name, err := redisClient.GetUserName(req.UserId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	user := &User{
		Id:        req.UserId,
		Name:      name,
		SessionId: req.SessionId,
	}
	if err := user.UpdateWordNote(req.Word, req.Note); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, nil)
}

func GetNote(c *gin.Context) {
	var req struct {
		UserId    string `json:"user_id"`
		SessionId string `json:"session_id"`
		Word      string `json:"word"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Word = c.PostForm("word")
		req.UserId = c.PostForm("user_id")
		req.SessionId = c.PostForm("session_id")
	}
	name, err := redisClient.GetUserName(req.UserId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	user := &User{
		Id:        req.UserId,
		Name:      name,
		SessionId: req.SessionId,
	}
	note, err := user.GetWordNote(req.Word)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, note)
}

func DeleteNote(c *gin.Context) {
	var req struct {
		UserId    string `json:"user_id"`
		SessionId string `json:"session_id"`
		Word      string `json:"word"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Word = c.PostForm("word")
		req.UserId = c.PostForm("user_id")
		req.SessionId = c.PostForm("session_id")
	}
	name, err := redisClient.GetUserName(req.UserId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	user := &User{
		Id:        req.UserId,
		Name:      name,
		SessionId: req.SessionId,
	}
	if err := user.DeleteWordNote(req.Word); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, nil)
}

func CreateNoteBookApi(c *gin.Context) {
	var req struct {
		SessionId string `json:"session_id"`
		BookName  string `json:"book_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.BookName = c.PostForm("book_name")
	}
	if err := CreateNoteBook(req.SessionId, req.BookName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, nil)
}
func AddWordToNotebookApi(c *gin.Context) {
	var req struct {
		SessionId string `json:"session_id"`
		BookName  string `json:"book_name"`
		Word      string `json:"word"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Word = c.PostForm("word")
	}
	if err := AddWordToNotebook(req.SessionId, req.Word, req.BookName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, nil)
}
func ReviewStart(c *gin.Context) {
	var req struct {
		SessionId string `json:"session_id"`
		BookName  string `json:"book_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.BookName = c.PostForm("book_name")
		req.SessionId = c.PostForm("session_id")
	}
	if req.SessionId == "" || req.BookName == "" {
		respondError(c, http.StatusBadRequest, "session_id and book_name are required")
		return
	}
	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	bookId, err := redisClient.GetUserBookId(uid, req.BookName)
	if err != nil {
		respondError(c, http.StatusBadRequest, "book not found for user")
		return
	}
	session, err := GetReview(uid, bookId, 10)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	reviewSessionsMu.Lock()
	reviewSessions[req.SessionId] = session
	reviewSessionsMu.Unlock()

	respondOK(c, session)
}

func getSessionID(c *gin.Context) (string, error) {
	sid := strings.TrimSpace(c.GetHeader("X-Session-Id"))
	if sid == "" {
		sid = strings.TrimSpace(c.Query("session_id"))
	}
	if sid == "" {
		return "", fmt.Errorf("session_id is empty")
	}
	return sid, nil
}

func getReviewSession(sessionID string) (*ReviewSession, error) {
	reviewSessionsMu.Lock()
	defer reviewSessionsMu.Unlock()
	session, ok := reviewSessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("review session not found")
	}
	return session, nil
}

func NextReview(c *gin.Context) {
	sessionID, err := getSessionID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	session, err := getReviewSession(sessionID)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	item := session.GetNext()
	if item == nil {
		respondOK(c, gin.H{"item": nil, "done": true})
		return
	}
	respondOK(c, gin.H{
		"index": session.CurrentIdx - 1,
		"item":  item,
		"done":  false,
	})
}

func SubmitReview(c *gin.Context) {
	sessionID, err := getSessionID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	session, err := getReviewSession(sessionID)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	var reqBody struct {
		Index   int  `json:"index"`
		Correct bool `json:"correct"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if reqBody.Index < 0 || reqBody.Index >= len(session.ReviewQueue) {
		respondError(c, http.StatusBadRequest, "invalid index")
		return
	}
	item := session.ReviewQueue[reqBody.Index]
	session.SubmitAnswer(item, reqBody.Correct)
	if session.Status == REVIEW_OVER {
		if err := session.saveProgress(); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		reviewSessionsMu.Lock()
		delete(reviewSessions, sessionID)
		reviewSessionsMu.Unlock()
	}
	respondOK(c, nil)
}

func respondOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, apiResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, apiResponse{
		Code:    status,
		Message: msg,
	})
}
