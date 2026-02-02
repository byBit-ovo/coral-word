package main

import (
	_"errors"
	"net/http"
	_"strings"
	"sync"
	"github.com/gin-gonic/gin"
)

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	reviewSessions   = map[string]*ReviewSession{}
	reviewSessionsMu sync.Mutex
)

func RunHTTPServer(addr string) error {
	router := gin.Default()

	router.POST("/login", Login)
	// router.POST("/api/register", Register)
	// router.GET("/api/word", WordQuery)

	// router.POST("/api/note", CreateNote)
	// router.PUT("/api/note", UpdateNote)
	// router.GET("/api/note", GetNote)
	// router.DELETE("/api/note", DeleteNote)

	// router.POST("/api/review/start", StartReview)
	// router.POST("/api/review/next", NextReview)
	// router.POST("/api/review/submit", SubmitReview)

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
	if err := user.userLogin(); err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	respondOK(c, gin.H{"session_id": user.SessionId})
}

// func Register(c *gin.Context) {
// 	var req struct {
// 		Name string `json:"name"`
// 		Pswd string `json:"pswd"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		req.Name = c.PostForm("name")
// 		req.Pswd = c.PostForm("pswd")
// 	}
// 	if req.Name == "" || req.Pswd == "" {
// 		respondError(c, http.StatusBadRequest, "name or password is empty")
// 		return
// 	}
// 	user := User{
// 		Name: req.Name,
// 		Pswd: req.Pswd,
// 	}
// 	if err := user.userRegister(); err != nil {
// 		respondError(c, http.StatusBadRequest, err.Error())
// 		return
// 	}
// 	err := user.userLogin()
// 	if err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, gin.H{"session_id": user.SessionId})
// }

// func WordQuery(c *gin.Context) {
// 	word := strings.TrimSpace(c.Query("word"))
// 	if word == "" {
// 		respondError(c, http.StatusBadRequest, "word is empty")
// 		return
// 	}
// 	wordDesc, err, _ := QueryWords(word)
// 	if err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, wordDesc)
// }

// func CreateNote(c *gin.Context) {
// 	user, err := getUserFromSession(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	var req struct {
// 		Word string `json:"word"`
// 		Note string `json:"note"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		req.Word = c.PostForm("word")
// 		req.Note = c.PostForm("note")
// 	}
// 	if req.Word == "" {
// 		respondError(c, http.StatusBadRequest, "word is empty")
// 		return
// 	}
// 	if err := user.CreateWordNote(req.Word, req.Note); err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, nil)
// }

// func UpdateNote(c *gin.Context) {
// 	user, err := getUserFromSession(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	var req struct {
// 		Word string `json:"word"`
// 		Note string `json:"note"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		req.Word = c.PostForm("word")
// 		req.Note = c.PostForm("note")
// 	}
// 	if req.Word == "" {
// 		respondError(c, http.StatusBadRequest, "word is empty")
// 		return
// 	}
// 	if err := user.UpdateWordNote(req.Word, req.Note); err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, nil)
// }

// func GetNote(c *gin.Context) {
// 	user, err := getUserFromSession(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	word := strings.TrimSpace(c.Query("word"))
// 	if word == "" {
// 		respondError(c, http.StatusBadRequest, "word is empty")
// 		return
// 	}
// 	note, err := user.GetWordNote(word)
// 	if err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, note)
// }

// func DeleteNote(c *gin.Context) {
// 	user, err := getUserFromSession(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	word := strings.TrimSpace(c.Query("word"))
// 	if word == "" {
// 		respondError(c, http.StatusBadRequest, "word is empty")
// 		return
// 	}
// 	if err := user.DeleteWordNote(word); err != nil {
// 		respondError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	respondOK(c, nil)
// }

// func StartReview(c *gin.Context) {
// 	user, err := getUserFromSession(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	var req struct {
// 		BookID string `json:"book_id"`
// 		Limit  int    `json:"limit"`
// 	}
// 	_ = c.ShouldBindJSON(&req)
// 	if req.BookID == "" {
// 		req.BookID, err = redisClient.GetUserBookId(user.Id, "我的生词本")
// 		if err != nil {
// 			respondError(c, http.StatusInternalServerError, err.Error())
// 			return
// 		}
// 	}
// 	if req.Limit <= 0 {
// 		req.Limit = 10
// 	}
// 	session, err := GetReview(user.Id, req.BookID, req.Limit)
// 	if err != nil {
// 		respondError(c, http.StatusBadRequest, err.Error())
// 		return
// 	}
// 	reviewSessionsMu.Lock()
// 	reviewSessions[user.SessionId] = session
// 	reviewSessionsMu.Unlock()

// 	item := session.GetNext()
// 	if item == nil {
// 		respondError(c, http.StatusBadRequest, NO_PENDING_REVIEWS)
// 		return
// 	}
// 	respondOK(c, gin.H{
// 		"index": session.CurrentIdx - 1,
// 		"item":  item,
// 	})
// }

// func NextReview(c *gin.Context) {
// 	sessionID, err := getSessionID(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	session, err := getReviewSession(sessionID)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	item := session.GetNext()
// 	if item == nil {
// 		respondOK(c, gin.H{"item": nil, "done": true})
// 		return
// 	}
// 	respondOK(c, gin.H{
// 		"index": session.CurrentIdx - 1,
// 		"item":  item,
// 	})
// }

// func SubmitReview(c *gin.Context) {
// 	sessionID, err := getSessionID(c)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	session, err := getReviewSession(sessionID)
// 	if err != nil {
// 		respondError(c, http.StatusUnauthorized, err.Error())
// 		return
// 	}
// 	var req struct {
// 		Index   int  `json:"index"`
// 		Correct bool `json:"correct"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		respondError(c, http.StatusBadRequest, "invalid request body")
// 		return
// 	}
// 	if req.Index < 0 || req.Index >= len(session.ReviewQueue) {
// 		respondError(c, http.StatusBadRequest, "invalid index")
// 		return
// 	}
// 	item := session.ReviewQueue[req.Index]
// 	session.SubmitAnswer(item, req.Correct)
// 	if session.Status == REVIEW_OVER {
// 		if err := session.saveProgress(); err != nil {
// 			respondError(c, http.StatusInternalServerError, err.Error())
// 			return
// 		}
// 		reviewSessionsMu.Lock()
// 		delete(reviewSessions, sessionID)
// 		reviewSessionsMu.Unlock()
// 	}
// 	respondOK(c, nil)
// }

// func getUserFromSession(c *gin.Context) (*User, error) {
// 	sid, err := getSessionID(c)
// 	if err != nil {
// 		return nil, err
// 	}
// 	uid, err := redisClient.GetUserSession(sid)
// 	if err != nil {
// 		return nil, errors.New("invalid session_id")
// 	}
// 	user, err := selectUserByID(uid)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.SessionId = sid
// 	return user, nil
// }

// func getReviewSession(sessionID string) (*ReviewSession, error) {
// 	reviewSessionsMu.Lock()
// 	defer reviewSessionsMu.Unlock()
// 	session, ok := reviewSessions[sessionID]
// 	if !ok {
// 		return nil, errors.New("review session not found")
// 	}
// 	return session, nil
// }

// func getSessionID(c *gin.Context) (string, error) {
// 	sid := strings.TrimSpace(c.GetHeader("X-Session-Id"))
// 	if sid == "" {
// 		sid = strings.TrimSpace(c.Query("session_id"))
// 	}
// 	if sid == "" {
// 		return "", errors.New("session_id is empty")
// 	}
// 	return sid, nil
// }

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
