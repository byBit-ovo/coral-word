package main

type WordNote struct{
	WordID int64
	UserID string
	NotebookID string
	Note string
}

func (wn *WordNote) CreateWordNote(wordID, userID, notebookID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
}