package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"github.com/google/uuid"
	_ "github.com/pingcap/log"
	_ "github.com/ydb-platform/ydb-go-sdk/v3/log"
)

type User struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Pswd      string `json:"pswd"`
	SessionId string
}

// if word not in database, query from llm and insert into database
func (user *User) CreateWordNote(wordName string, note string) error {
	word_desc, err := QueryWords(wordName)
	if err != nil {
		return err
	}
	wordNote := WordNote{
		WordID:   word_desc[wordName].WordID,
		UserID:   user.Id,
		UserName: user.Name,
		Note:     note,
		Selected: false,
	}
	return wordNote.CreateWordNote()
}

// test over
func (user *User) UpdateWordNote(wordName string, note string) error {
	word_desc, err := QueryWords(wordName)
	if err != nil {
		return err
	}
	wordNote := WordNote{
		WordID: word_desc[wordName].WordID,
		UserID: user.Id,
		Note:   note,
	}
	return wordNote.UpdateWordNote()
}

// test over
func (user *User) DeleteWordNote(wordName string) error {
	word_desc, err := QueryWords(wordName)
	if err != nil {
		return err
	}
	wordNote := WordNote{
		WordID:   word_desc[wordName].WordID,
		UserID:   user.Id,
		Note:     "",
		Selected: false,
	}
	return wordNote.DeleteWordNote()
}

// test over
func (user *User) GetWordNote(wordName string) (*WordNote, error) {
	word_desc, err := QueryWords(wordName)
	if err != nil {
		log.Println("QueryWords error:", err)
		return nil, err
	}
	wordNote := WordNote{
		WordID: word_desc[wordName].WordID,
		UserID: user.Id,
	}
	// fmt.Println(wordNote.WordID, wordNote.UserID)
	err = wordNote.GetWordNote()
	if err != nil {
		log.Println("GetWordNote error:", err)
		return nil, err
	}
	return &wordNote, nil
}

// test over
func (user *User) AppendWordNote(wordName string, note string) error {
	word_desc, err := QueryWords(wordName)
	if err != nil {
		return err
	}
	wordNote := WordNote{
		WordID: word_desc[wordName].WordID,
		UserID: user.Id,
	}
	return wordNote.AppendNote(note)
}

// function for administrator to set selected word note
// test over
func (user *User) SetSelectedWordNote(wordName string, selected bool) error {
	// get word_id from database
	word_desc, err := QueryWords(wordName)
	if err != nil {
		return err
	}
	wordNote := WordNote{
		WordID:   word_desc[wordName].WordID,
		UserID:   user.Id,
		Selected: selected,
	}
	return wordNote.SetSelectedWordNote(selected)
}

func (user *User) GetSelectedWordNotes(wordName string) ([]WordNote, error) {

	return GetSelectedWordNotes(wordName)
}

// test over
func (user *User) reviewWords() {
	Uniquewords := StartReview(user.SessionId)
	words := []string{}
	for word, _ := range Uniquewords {
		words = append(words, word)
	}
	article, err := GetArticleDesc(words)
	if err != nil {
		article.show()
	}
}

func (user *User) userRegister() error {
	new_user, _ := selectUser(user.Name)
	if new_user != nil {
		return fmt.Errorf("user:%s has registered, please login", user.Name)
	}
	new_user, err := insertUser(user.Name, user.Pswd)
	user.Id = new_user.Id
	if err != nil {
		return err
	}
	return nil

}
func (user *User) selectNoteBooks() (map[string]string, error) {

	rows, err := db.Query("select book_id,book_name from note_book where user_id=?", user.Id)
	if err != nil {
		return nil, err
	}
	books := map[string]string{}
	defer rows.Close()
	for rows.Next() {
		var book_name string
		var book_id string
		if err := rows.Scan(&book_id, &book_name); err != nil {
			return books, err
		}
		books[book_id] = book_name
	}
	if err := rows.Err(); err != nil {
		return books, err
	}
	return books, nil

}
func selectNoteWords(books map[string]string) (map[string][]int64, error) {
	noteWords := map[string][]int64{}
	for book_id, book_name := range books {
		rows, err := db.Query("select word_id from learning_record where book_id=?", book_id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var word_id int64
			if err := rows.Scan(&word_id); err != nil {
				rows.Close()
				return nil, err
			}
			noteWords[book_name] = append(noteWords[book_name], word_id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()

	}
	return noteWords, nil
}

// should be called when a new user login
// func listNoteWords(uid string) error {
// 	//book_id->book_name
// 	books, err := selectNoteBooks(uid)
// 	if err != nil {
// 		return err
// 	}
// 	for k, v := range books {
// 		// uid_bookname: book_id
// 		userBookToId[uid+"_"+v] = k
// 	}
// 	//book_name->[]word_ids
// 	noteWords, err := selectNoteWords(books)
// 	if err != nil {
// 		return err
// 	}
// 	userNoteWords[uid] = noteWords

//		var allWordIDs []int64
//		seen := make(map[int64]struct{})
//		for _, words := range userNoteWords[uid] {
//			for _, word_id := range words {
//				if _, ok := seen[word_id]; ok {
//					continue
//				}
//				seen[word_id] = struct{}{}
//				allWordIDs = append(allWordIDs, word_id)
//			}
//		}
//		if len(allWordIDs) == 0 {
//			return nil
//		}
//		wordMap, err := selectWordsByIds(allWordIDs...)
//		if err != nil {
//			return err
//		}
//		for wordID, wordDesc := range wordMap {
//			wordsPool[wordID] = wordDesc
//			wordNameToID[wordDesc.Word] = wordID
//		}
//		return nil
//	}
func (user *User) userLogin() error {
	u, err := selectUser(user.Name)
	if err != nil {
		return fmt.Errorf("User: %s doesn't exist", user.Name)
	}
	if user.Pswd != u.Pswd {
		return errors.New("incorrect password")
	}
	sessionId := uuid.New().String()
	err = redisClient.SetUserSession(sessionId, u.Id)
	user.SessionId = sessionId
	user.Id = u.Id
	if err != nil {
		return err
	}
	// listNoteWords(user.Id)
	return nil
}
func (user *User) userLogout() error {
	err := redisClient.DelUserSession(user.SessionId)
	if err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	return nil
}
func selectUser(name string) (*User, error) {
	row := db.QueryRow("select id, name, pswd from user where name=?", name)
	user := &User{}
	if err := row.Scan(&(user.Id), &(user.Name), &(user.Pswd)); err != nil {
		return nil, err
	}
	return user, nil
}

func selectUserByID(uid string) (*User, error) {
	row := db.QueryRow("select id, name, pswd from user where id=?", uid)
	user := &User{}
	if err := row.Scan(&(user.Id), &(user.Name), &(user.Pswd)); err != nil {
		return nil, err
	}
	return user, nil
}

func insertUser(name, pswd string) (*User, error) {
	tryUser, err := selectUser(name)
	if err != nil && err != sql.ErrNoRows {
		return nil, err // 查询出错
	}
	if tryUser != nil {
		return nil, fmt.Errorf("user already exists")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	id := uuid.New().String()
	_, err = tx.Exec("insert into user (id, name, pswd) values (?,?,?)", id, name, pswd)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &User{id, name, pswd, ""}, nil
}
