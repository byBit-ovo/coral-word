package main

import (
	_"os"
	_"log"
	_"bufio"
	_"context"
	_"errors"
)

//mysql, elasticsearch, redis
func sync() error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	rows, err := tx.Query("SELECT id FROM vocabulary")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		word, err := selectWordById(id)
		if err != nil {
			return err
		}
		err = IndexWordDesc(word)
		if err != nil {
			return err
		}
		if err = redisWordClient.HSet(word.Word,id); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// scale up words source
// func scaleUpWords(count int) error{
// 	file, err := os.OpenFile(os.Getenv("WORD_SOURCE_FILE"), os.O_RDONLY, 0644)
// 	if err != nil {
// 		log.Fatal("open file error:", err)
// 	}
// 	defer file.Close()
// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() && count > 0 {
// 		word := scanner.Text()
// 		err = redisWordClient.HSet(word,)
// 		if err != nil {
// 			return err
// 		}
// 		word_desc, err :=QueryWord(word)
// 		if err != nil{
// 			return err
// 		}
// 		err = IndexWordDesc(word_desc)
// 		count -= 1
// 	}
// 	return nil
// }

// func checkSyncAndSave()(map[int][]string, error){
// 	countInRedis, err := redisClient.SCard(context.Background(),"coral_word").Result()
// 	if err != nil{
// 		return nil, err
// 	}
// 	countInDB, err := db.Query("SELECT COUNT(*) FROM vocabulary")
// 	if err != nil{
// 		return nil, err
// 	}
// 	wordsEs, err := SearchAllWordIDs(500)
// 	if err != nil{
// 		return nil, err
// 	}
// 	return nil
// }
