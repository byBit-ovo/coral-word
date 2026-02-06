package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //如果字段是“空值”，就不要出现在 JSON 里。
}

func RunHTTPServer(addr string) error {
	router := gin.Default()

	router.POST("/coral/login", Login)
	router.POST("/coral/register", Register)
	router.GET("/coral/word", WordQuery)
	router.POST("/coral/create_note", CreateNote)
	router.PUT("/coral/update_note", UpdateNote)
	router.GET("/coral/get_note", GetNote)
	router.DELETE("/coral/delete_note", DeleteNote)
	router.POST("/coral/create_note_book", CreateNoteBookApi)
	router.POST("/coral/add_word_to_notebook", AddWordToNotebookApi)
	router.POST("/coral/review/start", ReviewStart)
	router.GET("/coral/review/next", NextReview)
	router.POST("/coral/review/submit", SubmitReview)

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
	// 输出完整 HTTP 请求报文到控制台
	// req := c.Request
	// // 请求行：Method RequestURI Proto
	// fmt.Println("-------- HTTP Request --------")
	// fmt.Printf("%s %s %s\n", req.Method, req.URL.RequestURI(), req.Proto)
	// for k, v := range req.Header {
	// 	for _, vv := range v {
	// 		fmt.Printf("%s: %s\n", k, vv)
	// 	}
	// }
	// if req.Body != nil {
		// body, _ := io.ReadAll(req.Body)
	// 	req.Body = io.NopCloser(bytes.NewBuffer(body)) // 恢复 Body 供后续使用
	// 	if len(body) > 0 {
	// 		fmt.Println()
	// 		fmt.Println(string(body))
	// 	}
	// }
	// fmt.Println("------------------------------")

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
	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}	
	user := &User{
		Id:        uid,
		SessionId: req.SessionId,
	}
	if err := user.CreateNoteBook(req.BookName); err != nil {
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
	uid, err := redisClient.GetUserSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	user := &User{
		Id:        uid,
		SessionId: req.SessionId,
	}
	if err := user.AddWordToNotebook(req.Word, req.BookName); err != nil {
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
	session, err := GetReview(uid, req.BookName, 10)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	// 存入 Redis（30 分钟过期）
	if err := redisClient.SetReviewSession(req.SessionId, session); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save session")
		return
	}
	respondOK(c, gin.H{
		"total": len(session.ReviewQueue),
		"msg":   "review session started",
	})
}

// NextReview 获取下一题（GET，session_id 在 Query 参数）
func NextReview(c *gin.Context) {
	// sessionID := strings.TrimSpace(c.Query("session_id"))
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		respondError(c, http.StatusBadRequest, "session_id is required")
		return
	}
	session, err := redisClient.GetReviewSession(sessionID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		respondError(c, http.StatusNotFound, "review session not found or expired")
		return
	}
	item, err := session.GetNext()
	if err != nil {

		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	// 更新 session 到 Redis（刷新 TTL）
	redisClient.SetReviewSession(sessionID, session)
	respondOK(c, gin.H{
		"next_index": session.CurrentIdx,
		"word":  item.WordDesc.Word,
		"done":  false,
	})
}

// SubmitReview 提交答案（POST，session_id 在请求体）
func SubmitReview(c *gin.Context) {
	var req struct {
		SessionId string `json:"session_id"`
		Index     int    `json:"index"`
		Correct   bool   `json:"correct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionId == "" {
		respondError(c, http.StatusBadRequest, "session_id is required")
		return
	}
	session, err := redisClient.GetReviewSession(req.SessionId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		respondError(c, http.StatusNotFound, "review session not found or expired")
		return
	}
	if req.Index < 0 || req.Index >= len(session.ReviewQueue) {
		respondError(c, http.StatusBadRequest, "invalid index")
		return
	}
	item := session.ReviewQueue[req.Index]
	session.SubmitAnswer(item, req.Correct)
	if session.Status == REVIEW_OVER {
		if err := session.saveProgress(); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		// 复习结束，删除 session
		redisClient.DelReviewSession(req.SessionId)
		respondOK(c, gin.H{"msg": "review completed"})
		return
	}
	// 更新 session 到 Redis
	redisClient.SetReviewSession(req.SessionId, session)
	respondOK(c, gin.H{"next_index": session.CurrentIdx, 
	"word": item.WordDesc, "done": false})
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
