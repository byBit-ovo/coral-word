package main

import(
	"database/sql"
)

var db *sql.DB
var userSession = map[string]string{}
var wordsPool = map[int64]*wordDesc{}

//						 uid       ntbn    wordIds
var userNoteWords = map[string]map[string][]int64{}

// uid_book_name -> book_id
var userBookToId = map[string]string{}

const (
	NO_PENDING_REVIEWS = "no pending reviews for today"
)
