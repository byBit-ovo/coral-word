package main

import (
	"database/sql"
	_"errors"
	_"fmt"
	"os"
	"strings"
	_"sort"
	_ "strconv"

	// "time"
	_ "github.com/go-sql-driver/mysql"
	_"github.com/google/uuid"
)


func InitSQL() error {
	var err error
	mysql_url := os.Getenv("MYSQL_URL")
	// 确保 DSN 包含 parseTime=true，以便自动解析时间字段
	if mysql_url != "" && !strings.Contains(mysql_url, "parseTime") {
		separator := "?"
		if strings.Contains(mysql_url, "?") {
			separator = "&"
		}
		mysql_url += separator + "parseTime=true&loc=Local"
	}
	db, err = sql.Open("mysql", mysql_url)
	if err != nil {
		return err
	}
	return nil
}
func minDistance(word1 string, word2 string) int {
    cache := make([][]int, len(word1)+1)
    for i,_ := range cache{
        cache[i] = make([]int, len(word2)+1)
        for j,_ := range cache[i]{
            cache[i][j] = -1
        }
    }
    var dfs func(int,int)int
    dfs = func(i int, j int) int{
        if i == len(word1){
            cache[i][j] = len(word2) - j
            return len(word2)-j
        }
        if j == len(word2){
            cache[i][j] = len(word1) - i
            return len(word1) - i
        }
        if cache[i][j] != -1{
            return cache[i][j]
        }
        if word1[i] == word2[j]{
            cache[i][j] = dfs(i+1,j+1)
            return cache[i][j]
        }

        cache[i][j] = min(
            // add
            1 + dfs(i,j+1),
            // delete
            1 + dfs(i+1,j),
            // replace
            1 + dfs(i+1, j+1),
        )
        return cache[i][j]
    }
    return dfs(0,0)
}

// 
func selectWordById(wordID int64)(w *wordDesc, err error){
    w = &wordDesc{}
    var tag int64
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }
    defer func() { 
		if err != nil{
			_ = tx.Rollback() 
		}
	}()

    // 查询主表
	source := 0
    row := tx.QueryRow("SELECT word, pronunciation, tag, source FROM vocabulary WHERE id = ?", wordID)
    if err = row.Scan(&w.Word, &w.Pronunciation, &tag,&source); err != nil {
        return nil, err
    }
	if err = aggWord(w,tx,wordID,tag);err != nil{
		return nil, err
	}
	w.WordID = wordID
	if err = tx.Commit(); err != nil {
        return nil,err
    }
	return w, nil
}

func selectWordByName(word string) (w *wordDesc, err error) {
    w = &wordDesc{}
    var tag int64
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }
    defer func() { 
		if err != nil{
			_ = tx.Rollback() 
		}
	}()
	source := 0
    // 查询主表
    row := tx.QueryRow("SELECT id, word, pronunciation, tag, source FROM vocabulary WHERE word = ?", word)
    if err = row.Scan(&w.WordID, &w.Word, &w.Pronunciation, &tag, &source); err != nil {
        return nil, err
    }
	if err = aggWord(w,tx,w.WordID,tag);err != nil{
		return nil, err
	}
	if err = tx.Commit(); err != nil {
        return nil,err
    }
	return w, nil

}
func aggWord(wordDesc *wordDesc, tx *sql.Tx, wordID int64, tag int64 )error{
	// updateQuery := fmt.Sprintf("update vocabulary set hit_count=hit_count+1 where word_id = '%d' ", wordID)
	// _, err := tx.Exec(updateQuery)
	_, err := tx.Exec("UPDATE vocabulary SET hit_count = hit_count + 1 WHERE id = ?", wordID)
	if err != nil{
		return err
	}
    wordDesc.Exam_tags = TagsFromMask(tag)

    // 查询 definitions
    definitions, err := queryDefinitions(tx, wordID)
    if err != nil {
        return err
    }
    wordDesc.Definitions = definitions

    // 查询 synonyms
    synonyms, err := queryStrings(tx, "SELECT syn FROM synonyms WHERE word_id = ?", wordID)
    if err != nil {
        return err
    }
    wordDesc.Synonyms = synonyms

    // 查询 derivatives
    derivatives, err := queryStrings(tx, "SELECT der FROM derivatives WHERE word_id = ?", wordID)
    if err != nil {
        return err
    }
    wordDesc.Derivatives = derivatives

    // 查询 example
    row := tx.QueryRow("SELECT sentence, translation FROM example WHERE word_id = ?", wordID)
    if err := row.Scan(&wordDesc.Example, &wordDesc.Example_cn); err != nil && err != sql.ErrNoRows {
        return err
    }

    // 查询 phrases
    phrases, err := queryPhrases(tx, wordID)
    if err != nil {
        return err
    }
    wordDesc.Phrases = phrases
    return nil
}
// 查询 definitions
func queryDefinitions(tx *sql.Tx, wordID int64) ([]Definition, error) {
    rows, err := tx.Query("SELECT pos, translation FROM vocabulary_cn WHERE word_id = ?", wordID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    definitionsMap := make(map[string][]string)
    for rows.Next() {
        var pos, translation string
        if err := rows.Scan(&pos, &translation); err != nil {
            return nil, err
        }
        definitionsMap[pos] = append(definitionsMap[pos], translation)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }

    var definitions []Definition
    for pos, translations := range definitionsMap {
        definitions = append(definitions, Definition{Pos: pos, Meanings: translations})
    }
    return definitions, nil
}

// 查询字符串列表（如 synonyms, derivatives）
func queryStrings(tx *sql.Tx, query string, args ...interface{}) ([]string, error) {
    rows, err := tx.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []string
    for rows.Next() {
        var value string
        if err := rows.Scan(&value); err != nil {
            return nil, err
        }
        results = append(results, value)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return results, nil
}

// 查询 phrases
func queryPhrases(tx *sql.Tx, wordID int64) ([]Phrase, error) {
    rows, err := tx.Query("SELECT phrase, translation FROM phrases WHERE word_id = ?", wordID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var phrases []Phrase
    for rows.Next() {
        var phrase, translation string
        if err := rows.Scan(&phrase, &translation); err != nil {
            return nil, err
        }
        phrases = append(phrases, Phrase{Example: phrase, Example_cn: translation})
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return phrases, nil
}






// func selectWord(word string)(*wordDesc, error){
// 	var word_id int32
// 	word_desc := wordDesc{}
// 	var tag int32
// 	tx, err := db.Begin()
// 	if err != nil{
// 		return nil, err
// 	}
// 	defer func() {_ = tx.Rollback() }()
// 	row := tx.QueryRow("select id, word, pronunciation, tag from vocabulary where word=?",word)
// 	err = row.Scan(&word_id, &word_desc.Word,&word_desc.Pronunciation,&tag)
// 	if err != nil{
// 		return nil, err
// 	}
// 	word_desc.Exam_tags = TagsFromMask(tag)
// 	rows, err := tx.Query("select pos, translation from vocabulary_cn where word_id=?",word_id)
// 	if err != nil{
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	var definitions = make(map[string][]string)
// 	for rows.Next(){
// 		var pos string
// 		var trans string
// 		if err := rows.Scan(&pos, &trans); err != nil {
//         	return nil, err
//     	}
// 		definitions[pos] = append(definitions[pos], trans)
// 	}
// 	if err := rows.Err(); err != nil {
//         return nil, err
//     }
// 	for k,v := range definitions{
// 		word_desc.Definitions = append(word_desc.Definitions, Definition{k,v})
// 	}
// 	rows, err = tx.Query("select syn from synonyms where word_id = ?", word_id)
// 	if err != nil{
// 		return nil, err
// 	}
// 	for rows.Next(){
// 		var syn string
// 		if err = rows.Scan(&syn);err != nil{
// 			return nil, err
// 		}
// 		word_desc.Synonyms = append(word_desc.Synonyms, syn)
// 	}
// 	rows, err = tx.Query("select der from derivatives where word_id = ?", word_id)
// 	if err != nil{
// 		return nil, err
// 	}
// 	for rows.Next(){
// 		var der string
// 		if err = rows.Scan(&der); err != nil{
// 			return nil, err
// 		}
// 		word_desc.Derivatives = append(word_desc.Derivatives, der)
// 	}
// 	if err := rows.Err(); err != nil {
//         return nil, err
//     }
// 	row = tx.QueryRow("select sentence, translation from example where word_id=?",word_id)
// 	if err = row.Scan(&word_desc.Example, &word_desc.Example_cn); err != nil{
// 		return nil, err
// 	}
// 	rows, err = tx.Query("select phrase, translation from phrases where word_id=? ",word_id)
// 	if err != nil{
// 		return nil, err
// 	}
// 	for rows.Next(){
// 		var phrase, translation string
// 		if rows.Scan(&phrase, &translation) != nil{
// 			return nil, err
// 		}
// 		word_desc.Phrases = append(word_desc.Phrases, Phrase{phrase, translation})
// 	}
// 	if err := rows.Err(); err != nil {
//         return nil, err
//     }

// 	return &word_desc, tx.Commit()
// }