package main

import (
	"database/sql"
	"errors"
)

type WordNote struct {
	WordID   int64
	UserID   string
	UserName string
	Note     string
	Selected bool
}

func (wn *WordNote) CreateWordNote() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.Exec("insert into word_note (word_id, user_id, note,selected) values (?, ?, ?,?)", wn.WordID, wn.UserID, wn.Note, wn.Selected)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// word_id and user_id are required, only note will be updated
func (wn *WordNote) UpdateWordNote() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var result sql.Result
	result, err = tx.Exec("update word_note set note = ? where word_id = ? and user_id = ?", wn.Note, wn.WordID, wn.UserID)
	if err != nil {
		return err
	}
	var affected int64
	affected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("no rows updated for word_note")
	}
	return tx.Commit()
}

// word_id and user_id are required, only note will be updated
func (wn *WordNote) AppendNote(note string) error {
	err := wn.GetWordNote()
	if err != nil {
		return err
	}
	wn.Note += "\n" + note
	return wn.UpdateWordNote()
}
func (wn *WordNote) DeleteWordNote() error {
	wn.Note = ""
	wn.Selected = false
	return wn.UpdateWordNote()
}

// word_id and user_id are required, note will be set into caller
func (wn *WordNote) GetWordNote() error {

	row := db.QueryRow("select note, selected from word_note where word_id = ? and user_id = ?", wn.WordID, wn.UserID)
	return row.Scan(&wn.Note, &wn.Selected)
}

//user_id and word_id are required, only selected will be updated
func (wn *WordNote) SetSelectedWordNote(selected bool) error{
	wn.Selected = selected
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.Exec("update word_note set selected = ? where word_id = ? and user_id = ?", wn.Selected, wn.WordID, wn.UserID)
	if err != nil {
		return err
	}
	return tx.Commit()
}


func GetSelectedWordNotes(wordName string) ([]WordNote, error) {
	wordNotes := []WordNote{}
	wordID, ok := wordNameToID[wordName]
	if !ok {
		word_desc, err := QueryWord(wordName)
		if err != nil {
			return nil, err
		}
		wordID = word_desc.WordID
	}
	rows, err := db.Query("select user_id, note from word_note where word_id = ? and selected = true", wordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var wordNote WordNote
		if err := rows.Scan(&wordNote.UserID, &wordNote.Note); err != nil {
			return nil, err
		}
		err = db.QueryRow("select name from user where id = ?", wordNote.UserID).Scan(&wordNote.UserName)
		if err != nil {
			return nil, err
		}
		wordNotes = append(wordNotes, wordNote)
	}
	return wordNotes, nil
}


