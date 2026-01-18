package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/byBit-ovo/coral_word/llm"
	"github.com/google/uuid"
	_ "github.com/pingcap/log"
	_ "github.com/ydb-platform/ydb-go-sdk/v3/log"
)


type User struct{
	Id string `json:"id"`
	Name string `json:"name"`
	Pswd string `json:"pswd"`
	SessionId 	string
}

func (user *User) reviewWords(){
	Uniquewords := StartReview(user.SessionId)
	words := []string{}
	for word,_ := range Uniquewords{
		words = append(words, word)
	}
	article, err := GetArticleDesc(words)
	if err != nil{
		article.show()
	}
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
func selectNoteBooks(uid string)(map[string]string, error){
	
	rows, err := db.Query("select book_id,book_name from note_book where user_id=?",uid)
	if err !=nil{
		return nil, err
	}
	books := map[string]string{}
	defer rows.Close()
	for rows.Next(){
		var book_name string
		var book_id string
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
func selectNoteWords(books map[string]string)(map[string][]int64, error){
	noteWords := map[string][]int64{}
	for book_id,book_name := range books{
		rows, err := db.Query("select word_id from learning_record where book_id=?",book_id)
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
	for k,v := range books{
		// uid_bookname: book_id
		userBookToId[uid+"_"+v] = k
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
func userLogin(name, pswd string)(*User,error){
	user, err:= selectUser(name)
	if err != nil{
		return nil,fmt.Errorf("User: %s doesn't exist", name)
	}
	if user.Pswd != pswd{
		return nil, errors.New("incorrect password")
	}
	sessionId := uuid.New().String()
	userSession[sessionId] = user.Id
	user.SessionId = sessionId
	listNoteWords(user.Id)
	
	// fmt.Println("Login successfully!")
	return user, nil
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
	return &User{id,name,pswd,""}, nil
}





// func dailyCheckIn(uid string) error {
// 	var books map[string][]int64
// 	var ok bool
// 	if books, ok = userNoteWords[uid]; ok==false{
// 		fmt.Println("尚未创建笔记本!")
// 		return errors.New("用户尚未创建任何笔记本!")
// 	}
// 	fmt.Println("请选择要复习的笔记本:")
// 	for k,_ := range books{
// 		fmt.Print(k, " ")
// 	}
// 	var choice string
// 	fmt.Scan(&choice)
	
	
// }