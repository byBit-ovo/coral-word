package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)


func testGin() {
	router := gin.Default()
	router.GET("/search/:word", func(c *gin.Context) {
		word := c.Param("word")
		word_desc, err,_ := QueryWords(word)
		if err != nil {
			// 返回错误响应
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to query word",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, word_desc)
	})
	router.Run()
}
