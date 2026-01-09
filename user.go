package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	_"github.com/pingcap/log"
	_"github.com/ydb-platform/ydb-go-sdk/v3/log"
)

var userSession = map[string]string{}
var wordsPool = map[int64]*wordDesc{}

var userNoteWords = map[string]map[string][]int64{}
//						  uid       ntbn     wordId
type User struct{
	Id string `json:"id"`
	Name string `json:"name"`
	Pswd string `json:"pswd"`
}

func userRegister(name, pswd string)(*User, error){
	user, _:= selectUser(name)
	if user != nil{
		return user, fmt.Errorf("user:%s has registered, please login", name)
	}
	user, err := insertUser(name, pswd)
	if err != nil{
		return nil, err
	}
	return user, nil

}
func selectNoteBooks(uid string)(map[int64]string, error){
	
	rows, err := db.Query("select book_id,book_name from note_book where user_id=?",uid)
	if err !=nil{
		return nil, err
	}
	books := map[int64]string{}
	defer rows.Close()
	for rows.Next(){
		var book_name string
		var book_id int64
		if err:=rows.Scan(&book_id,&book_name); err !=nil{
			return books,err
		}
		books[book_id] = book_name
	}
	if err := rows.Err() ;err != nil{
		return books,err
	}
	return books, nil

}
func selectNoteWords(books map[int64]string)(map[string][]int64, error){
	noteWords := map[string][]int64{}
	for book_id,book_name := range books{
		rows, err := db.Query("select word_id from note_word where book_id=?",book_id)
		if err !=nil{
			return nil, err
		}
		for rows.Next(){
			var word_id int64
			if err:=rows.Scan(&word_id);err != nil{
				rows.Close()
				return nil,err
			}
			noteWords[book_name] = append(noteWords[book_name], word_id)
		}
		if err:=rows.Err();err !=nil{
			rows.Close()
			return nil, err
		}
		rows.Close()

	}
	return noteWords, nil
}
// should be called when a new user login 
func listNoteWords(uid string)error{
	//book_id->book_name
	books, err := selectNoteBooks(uid)
	if err != nil{
		return err
	}
	//book_name->[]word_ids
	noteWords, err := selectNoteWords(books)
	if err != nil{
		return err
	}
	userNoteWords[uid] = noteWords

	for _,words := range userNoteWords[uid]{
		
		for _,word_id := range words{
			word_desc, err:= selectWordById(word_id)
			if err != nil{
				fmt.Println("select word error:", word_id,err.Error())
				continue
			}
			wordsPool[word_id] = word_desc
		}
	}
	return nil
}
func userLogin(name, pswd string)(string,error){
	user, err:= selectUser(name)
	if err != nil{
		return "",fmt.Errorf("User: %s doesn't exist", name)
	}
	if user.Pswd != pswd{
		return "", errors.New("incorrect password")
	}
	sessionId := uuid.New().String()
	userSession[sessionId] = user.Id
	listNoteWords(user.Id)
	return sessionId, nil
}

func selectUser(name string) (*User, error){
	row := db.QueryRow("select id, name, pswd from user where name=?",name)
	user := &User{}
	if err := row.Scan(&(user.Id), &(user.Name), &(user.Pswd)); err != nil{
		return nil, err
	}
	return user, nil
}

func insertUser(name, pswd string) (*User, error){
	tryUser, err := selectUser(name)
	if err != nil && err != sql.ErrNoRows {
		return nil, err // 查询出错
	}
	if tryUser != nil {
		return nil, fmt.Errorf("user already exists")
	}
	tx, err := db.Begin()
	if err != nil{
		return nil, err
	}
	defer func(){
		if err != nil{
			_ = tx.Rollback()
		}
	}()
	id := uuid.New().String()
	_, err = tx.Exec("insert into user (id, name, pswd) values (?,?,?)",id ,name, pswd)
	if err != nil{
		return nil, err
	}
	if err = tx.Commit(); err != nil{
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &User{id,name,pswd}, nil
}



func createNoteBook(session string, bookName string) (err error){
	uid,ok := userSession[session]
	if ok == false{
		return fmt.Errorf("user isn't logged in !")
	}
	tx, err := db.Begin()
	if err != nil{
		return err
	}
	defer func(){
		if err != nil{
			_ = tx.Rollback()
		}
	}()
	_, err = tx.Exec("insert into note_book (book_name, user_id) values (?,?)", bookName,uid)
	if err != nil{
		return fmt.Errorf("failed to insert notebook: %w", err)
	}
	return tx.Commit()
}

func AddWordToNotebook(session, word, noteBookName string) (err error){
	uid,ok := userSession[session]
	if ok == false{
		return fmt.Errorf("user isn't logged in !")
	}
	tx, err := db.Begin()
	if err != nil{
		return err
	}
	defer func(){
		if err != nil{
			_ = tx.Rollback()
		}
	}()
	row := tx.QueryRow("select id from vocabulary where word = ?", word)
	var wordId int64
	if err:=row.Scan(&wordId); err != nil{
		return err
	}
	row = tx.QueryRow("select book_id from note_book where book_name=? and user_id=?", noteBookName, uid)
	var bookId int64
	if err:=row.Scan(&bookId); err != nil{
		return err
	}
	_, err = tx.Exec("insert into note_word (book_id, word_id) values (?,?)", bookId, wordId)
	if err != nil{
		return err
	}
	return tx.Commit()
}