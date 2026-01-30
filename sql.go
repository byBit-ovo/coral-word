package main

import (
	"database/sql"
	_ "errors"
	"fmt"
	"os"
	_ "sort"
	_ "strconv"
	"strings"

	// "time"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
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
	for i, _ := range cache {
		cache[i] = make([]int, len(word2)+1)
		for j, _ := range cache[i] {
			cache[i][j] = -1
		}
	}
	var dfs func(int, int) int
	dfs = func(i int, j int) int {
		if i == len(word1) {
			cache[i][j] = len(word2) - j
			return len(word2) - j
		}
		if j == len(word2) {
			cache[i][j] = len(word1) - i
			return len(word1) - i
		}
		if cache[i][j] != -1 {
			return cache[i][j]
		}
		if word1[i] == word2[j] {
			cache[i][j] = dfs(i+1, j+1)
			return cache[i][j]
		}

		cache[i][j] = min(
			// add
			1+dfs(i, j+1),
			// delete
			1+dfs(i+1, j),
			// replace
			1+dfs(i+1, j+1),
		)
		return cache[i][j]
	}
	return dfs(0, 0)
}

func selectWordsByIds(wordIds ...int64) (map[int64]*wordDesc, error) {
	if len(wordIds) == 0 {
		return map[int64]*wordDesc{}, nil
	}
	placeholders := strings.Repeat("?,", len(wordIds)-1) + "?"
	args := make([]interface{}, len(wordIds))
	for i, id := range wordIds {
		args[i] = id
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

	rows, err := tx.Query(fmt.Sprintf("SELECT id, word, pronunciation, tag, source FROM vocabulary WHERE id IN (%s)", placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wordsByID := make(map[int64]*wordDesc)
	for rows.Next() {
		w := &wordDesc{}
		var tag int64
		if err := rows.Scan(&w.WordID, &w.Word, &w.Pronunciation, &tag, &w.Source); err != nil {
			return nil, err
		}
		wordsByID[w.WordID] = w
		w.Exam_tags = TagsFromMask(tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err = aggWords(tx, wordsByID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return wordsByID, nil
}

func selectWordsByNames(words ...string) (map[string]*wordDesc, error) {
	if len(words) == 0 {
		return map[string]*wordDesc{}, nil
	}
	placeholders := strings.Repeat("?,", len(words)-1) + "?"
	args := make([]interface{}, len(words))
	for i, word := range words {
		args[i] = word
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

	rows, err := tx.Query(fmt.Sprintf("SELECT id, word, pronunciation, tag, source FROM vocabulary WHERE word IN (%s)", placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wordsByName := make(map[string]*wordDesc)
	for rows.Next() {
		w := &wordDesc{}
		var tag int64
		if err := rows.Scan(&w.WordID, &w.Word, &w.Pronunciation, &tag, &w.Source); err != nil {
			return nil, err
		}
		wordsByName[w.Word] = w
		w.Exam_tags = TagsFromMask(tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	wordsByID := make(map[int64]*wordDesc, len(wordsByName))
	for _, w := range wordsByName {
		wordsByID[w.WordID] = w
	}

	if err = aggWords(tx, wordsByID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return wordsByName, nil
}

func aggWords(tx *sql.Tx, wordsByID map[int64]*wordDesc) error {
	if len(wordsByID) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(wordsByID)-1) + "?"
	args := make([]interface{}, len(wordsByID))
	idx := 0
	for i, _ := range wordsByID {
		args[idx] = i
		idx += 1
	}
	if _, err := tx.Exec("UPDATE vocabulary SET hit_count = hit_count + 1 WHERE id IN ("+placeholders+")", args...); err != nil {
		return err
	}

	defRows, err := tx.Query("SELECT word_id, pos, translation FROM vocabulary_cn WHERE word_id IN ("+placeholders+")", args...)
	if err != nil {
		return err
	}
	defMap := make(map[int64]map[string][]string)
	for defRows.Next() {
		var id int64
		var pos, translation string
		if err := defRows.Scan(&id, &pos, &translation); err != nil {
			defRows.Close()
			return err
		}
		if defMap[id] == nil {
			defMap[id] = make(map[string][]string)
		}
		defMap[id][pos] = append(defMap[id][pos], translation)
	}
	if err := defRows.Err(); err != nil {
		defRows.Close()
		return err
	}
	defRows.Close()

	synRows, err := tx.Query("SELECT word_id, syn FROM synonyms WHERE word_id IN ("+placeholders+")", args...)
	if err != nil {
		return err
	}
	synMap := make(map[int64][]string)
	for synRows.Next() {
		var id int64
		var syn string
		if err := synRows.Scan(&id, &syn); err != nil {
			synRows.Close()
			return err
		}
		synMap[id] = append(synMap[id], syn)
	}
	if err := synRows.Err(); err != nil {
		synRows.Close()
		return err
	}
	synRows.Close()

	derRows, err := tx.Query("SELECT word_id, der FROM derivatives WHERE word_id IN ("+placeholders+")", args...)
	if err != nil {
		return err
	}
	derMap := make(map[int64][]string)
	for derRows.Next() {
		var id int64
		var der string
		if err := derRows.Scan(&id, &der); err != nil {
			derRows.Close()
			return err
		}
		derMap[id] = append(derMap[id], der)
	}
	if err := derRows.Err(); err != nil {
		derRows.Close()
		return err
	}
	derRows.Close()

	exRows, err := tx.Query("SELECT word_id, sentence, translation FROM example WHERE word_id IN ("+placeholders+")", args...)
	if err != nil {
		return err
	}
	type examplePair struct {
		example   string
		exampleCn string
	}
	exMap := make(map[int64]examplePair)
	for exRows.Next() {
		var id int64
		var sentence, translation string
		if err := exRows.Scan(&id, &sentence, &translation); err != nil {
			exRows.Close()
			return err
		}
		exMap[id] = examplePair{example: sentence, exampleCn: translation}
	}
	if err := exRows.Err(); err != nil {
		exRows.Close()
		return err
	}
	exRows.Close()

	phraseRows, err := tx.Query("SELECT word_id, phrase, translation FROM phrases WHERE word_id IN ("+placeholders+")", args...)
	if err != nil {
		return err
	}
	phraseMap := make(map[int64][]Phrase)
	for phraseRows.Next() {
		var id int64
		var phrase, translation string
		if err := phraseRows.Scan(&id, &phrase, &translation); err != nil {
			phraseRows.Close()
			return err
		}
		phraseMap[id] = append(phraseMap[id], Phrase{Example: phrase, Example_cn: translation})
	}
	if err := phraseRows.Err(); err != nil {
		phraseRows.Close()
		return err
	}
	phraseRows.Close()

	for id, w := range wordsByID {
		defPosMap := defMap[id]
		if len(defPosMap) > 0 {
			defs := make([]Definition, 0, len(defPosMap))
			for pos, translations := range defPosMap {
				defs = append(defs, Definition{Pos: pos, Meanings: translations})
			}
			w.Definitions = defs
		}
		w.Synonyms = synMap[id]
		w.Derivatives = derMap[id]
		if ex, ok := exMap[id]; ok {
			w.Example = ex.example
			w.Example_cn = ex.exampleCn
		}
		w.Phrases = phraseMap[id]
	}
	return nil
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
