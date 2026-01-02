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
	var tag int32
	tx, err := db.Begin()
	if err != nil{
		return nil, err
	}
	defer func() {_ = tx.Rollback() }()
	row := tx.QueryRow("select id, word, pronunciation, tag from vocabulary where word=?",word)
	err = row.Scan(&word_id, &word_desc.Word,&word_desc.Pronunciation,&tag)
	if err != nil{
		return nil, err
	}
	word_desc.Exam_tags = TagsFromMask(tag)
	rows, err := tx.Query("select pos, translation from vocabulary_cn where word_id=?",word_id)
	if err != nil{
		return nil, err
	}
	defer rows.Close()
	var definitions = make(map[string][]string)
	for rows.Next(){
		var pos string
		var trans string
		if err := rows.Scan(&pos, &trans); err != nil {
        	return nil, err
    	}
		definitions[pos] = append(definitions[pos], trans)
	}
	if err := rows.Err(); err != nil {
        return nil, err
    }
	for k,v := range definitions{
		word_desc.Definitions = append(word_desc.Definitions, Definition{k,v})
	}
	rows, err = tx.Query("select syn from synonyms where word_id = ?", word_id)
	if err != nil{
		return nil, err
	}
	for rows.Next(){
		var syn string
		if rows.Scan(&syn) != nil{
			return nil, err
		}
		word_desc.Synonyms = append(word_desc.Synonyms, syn)
	}
	rows, err = tx.Query("select der from derivatives where word_id = ?", word_id)
	if err != nil{
		return nil, err
	}
	for rows.Next(){
		var der string
		if rows.Scan(&der) != nil{
			return nil, err
		}
		word_desc.Derivatives = append(word_desc.Derivatives, der)
	}
	if err := rows.Err(); err != nil {
        return nil, err
    }
	row = tx.QueryRow("select sentence, translation from example where word_id = ?",word_id)
	if err = row.Scan(&word_desc.Example, &word_desc.Example_cn); err != nil{
		return nil, err
	}

	return &word_desc, tx.Commit()
}

func insertWord(word *wordDesc)error{
	tags := aggregateTags(word.Exam_tags)
	tx, err := db.Begin()
	if err != nil {
    	return err
	}
	defer func(){ _ = tx.Rollback()}()
	res, err := tx.Exec(`insert into vocabulary (word, pronunciation, tag) values (?,?,?)`, word.Word, word.Pronunciation, tags)
	if err != nil {
    	return err
	}
	word_id, err := res.LastInsertId()
	if err != nil{
		return err
	}
	for _, def := range word.Definitions{
		for _,tr := range def.Meanings{
			res, err = tx.Exec(`insert into vocabulary_cn (word_id, translation, pos) values (?,?,?)`,word_id,tr,def.Pos)
			if err != nil{
				return err
			}
		}

	}
	for _,der := range word.Derivatives{
		res, err = tx.Exec(`insert into derivatives (word_id, der) values (?,?)`,word_id, der)
		if err != nil{
			return err
		}
	}
	for _, syn := range word.Synonyms{
		res, err = tx.Exec("insert into synonyms (word_id, syn) values (?, ?)", word_id, syn)
		if err != nil{
			return err
		}
	}
	res, err = tx.Exec("insert into example (word_id, sentence, translation) values (?,?,?)", word_id, word.Example,word.Example_cn)
	if err != nil{
		return err
	}
	for _, phrase := range word.Phrases{
		res, err = tx.Exec("insert into phrases (word_id, phrase, translation) values (?,?,?)", word_id, phrase.Example,phrase.Example_cn)
		if err != nil{
			return err
		}
	}
	return tx.Commit()
}