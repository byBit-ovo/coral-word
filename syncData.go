package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// mysql, elasticsearch, redis

func syncMissingFromLogs() error {
	if err := syncMissingInEs("log/missInEs.log"); err != nil {
		return err
	}

	if err := syncMissingInRedis("log/missInRedis.log"); err != nil {
		return err
	}

	if err := clearFile("log/missInEs.log"); err != nil {
		return err
	}
	if err := clearFile("log/missInRedis.log"); err != nil {
		return err
	}
	file, _ := os.OpenFile("log/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	logger := log.New(file, "", log.LstdFlags)
	logger.Println("syncMissingFromLogs done")
	return nil
}

func clearFile(path string) error {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	
	return file.Close()
}

func syncMissingInEs(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		id, _, err := parseMissingLogLine(line)
		if err != nil {
			log.Println("parse missInEs line error:", err)
			continue
		}
		wordDesc, err := selectWordById(id)
		if err != nil {
			log.Println("selectWordById error:", err)
			continue
		}
		if err := esClient.IndexWordDesc(wordDesc); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func syncMissingInRedis(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		id, word, err := parseMissingLogLine(line)
		if err != nil {
			log.Println("parse missInRedis line error:", err)
			continue
		}
		if word == "" || word == "-" {
			log.Println("missInRedis word empty for id:", id)
			continue
		}
		if err := redisWordClient.HSet(word, id); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func parseMissingLogLine(line string) (int64, string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, "", fmt.Errorf("invalid line: %s", line)
	}
	idStr := strings.TrimPrefix(fields[0], "id=")
	wordStr := strings.TrimPrefix(fields[1], "word=")
	if idStr == fields[0] || wordStr == fields[1] {
		return 0, "", fmt.Errorf("invalid line: %s", line)
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, "", err
	}
	return id, wordStr, nil
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

func checkSyncLog() {
	file, _ := os.OpenFile("log/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	logger := log.New(file, "", log.LstdFlags)
	logger.Println("checkSyncAndSave start")
	wordsInRedis, err := redisWordClient.HGetAll()
	if err != nil {
		logger.Println("redisWordClient.HGetAll error:", err)
		return
	}
	rows, err := db.Query("SELECT id,word FROM vocabulary")
	if err != nil {
		logger.Println("db.Query error:", err)
		return
	}
	wordsInMysql := make(map[int64]string)
	err = func() error {
		for rows.Next() {
			var id int64
			var word string
			err = rows.Scan(&id, &word)
			if err != nil {
				return err
			}
			wordsInMysql[id] = word
		}
		return rows.Err()
	}()
	if err != nil {
		logger.Println("words in mysql error:", err)
		return
	}
	wordsInEs, err := esClient.SearchAllWordIDs(500)
	if err != nil {
		logger.Println("words in es error:", err)
		return
	}

	redisIDToWord := make(map[int64]string)
	for word, idStr := range wordsInRedis {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			logger.Println("words in redis error:", err)
			continue
		}
		redisIDToWord[id] = word
	}

	unionIDs := make(map[int64]string)
	for id := range wordsInMysql {
		unionIDs[id] = wordsInMysql[id]
	}
	for id := range wordsInEs {
		unionIDs[id] = wordsInEs[id]
	}
	for id := range redisIDToWord {
		unionIDs[id] = redisIDToWord[id]
	}

	missing := make(map[int64][]string)
	for id := range unionIDs {
		missingSources := make([]string, 0, 3)
		if _, ok := wordsInMysql[id]; ok == false {
			missingSources = append(missingSources, "mysql")
		}
		if _, ok := wordsInEs[id]; ok == false {
			missingSources = append(missingSources, "es")
		}
		if _, ok := redisIDToWord[id]; ok == false {
			missingSources = append(missingSources, "redis")
		}
		if len(missingSources) > 0 {
			missing[id] = missingSources
		}
	}

	if len(missing) == 0 {
		logger.Println("Words are all synced")
	} else {
		logger.Println("Words are not synced, details are in missingWord")
	}
	esFile, err := os.OpenFile("log/missInEs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open es.log error:", err)
		return
	}
	defer esFile.Close()
	mysqlFile, err := os.OpenFile("log/missInMysql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open mysql.log error:", err)
		return
	}
	defer mysqlFile.Close()
	redisFile, err := os.OpenFile("log/missInRedis.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open redis.log error:", err)
		return
	}
	defer redisFile.Close()
	for id, sources := range missing {
		word := unionIDs[id]
		if word == "" {
			word = "-"
		}
		for _, source := range sources {
			switch source {
			case "es":
				esFile.WriteString(fmt.Sprintf("id=%d word=%s\n", id, word))
			case "mysql":
				mysqlFile.WriteString(fmt.Sprintf("id=%d word=%s\n", id, word))
			case "redis":
				redisFile.WriteString(fmt.Sprintf("id=%d word=%s\n", id, word))
			}
		}
	}
}
