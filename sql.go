package main
import (
	"database/sql"
	// "time"

	_"github.com/go-sql-driver/mysql"
)


var db *sql.DB
func InitSQL() error {
	var err error
	db, err = sql.Open("mysql", "root:200533@/coral_word")
	if err != nil{
		return err
	}
	return nil
}

func selectWord(word string)(*wordDesc, error){
	var word_id int32
	word_desc := wordDesc{}
	tag := 0
	row := db.QueryRow("select id, word, pronunciation, tag from vocabulary where word=?",word)
	err := row.Scan(&word_id, &word_desc.Word,&word_desc.Pronunciation,&tag)
	if err != nil{
		return nil, err
	}
	return &word_desc, nil
}

func insertWord(word *wordDesc)error{
	tags := aggregateTags(word.Exam_tags)
	tx, err := db.Begin()
	if err != nil {
    	return err
	}
	defer tx.Rollback()
	res, err := db.Exec(`insert into vocabulary (word, pronunciation, tag) values (?,?,?)`, word.Word, word.Pronunciation, tags)
	if err != nil {
    	return err
	}
	word_id, err := res.LastInsertId()
	if err != nil{
		return err
	}
	for _, def := range word.Definitions{
		for _,tr := range def.Meanings{
			res, err = db.Exec(`insert into vocabulary_cn (word_id, translation, pos) values (?,?,?)`,word_id,tr,def.Pos)
		}
		if err != nil{
			return err
		}
	}
	for _,der := range word.Derivatives{
		res, err = db.Exec(`insert into derivatives (word_id, der) values (?,?)`,word_id, der)
		if err != nil{
			return err
		}
	}
	for _, syn := range word.Synonyms{
		res, err = db.Exec("insert into synonyms (word_id, syn) values (?, ?)", word_id, syn)
	}
	res, err = db.Exec("insert into example (word_id, sentence, translation) values (?,?,?)", word_id, word.Example,word.Example_cn)
	if err != nil{
		return err
	}
	for _, phrase := range word.Phrases{
		res, err = db.Exec("insert into phrases (word_id, phrase, translation) values (?,?.?)", word_id, phrase.Example,phrase.Example_cn)
	}
	return tx.Commit()
}