package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
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
	var ids []int64
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
		ids = append(ids, id)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	wordMap, err := selectWordsByIds(ids...)
	if err != nil {
		return err
	}
	for _, id := range ids {
		wordDesc, ok := wordMap[id]
		if !ok {
			log.Println("selectWordsByIds missing id:", id)
			continue
		}
		if err := esClient.IndexWordDesc(wordDesc); err != nil {
			return err
		}
	}
	return nil
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
		if err := redisClient.HSetWord(word, id); err != nil {
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



func scaleFromGoogle100000(count int) (map[string]*wordDesc, error, []string) {
	words, err := readFromGoogle100000(count)
	if err != nil {
		return nil, err, nil
	}
	return scaleUpWords(LLMPool, words...)	
}	


func readFromGoogle100000(count int) ([]string, error) {
	file, err := os.OpenFile(os.Getenv("WORD_SOURCE_FILE"), os.O_RDONLY, 0644)
	if err != nil {
		log.Println("open file error:", err)
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		log.Println("scanner error:", err)
		return nil, err
	}	
	words := make([]string,0)
	for scanner.Scan() && count != 0{
		word := scanner.Text()
		if _, err := redisClient.HGetWord(word);err == nil{
			continue
		}
		words = append(words, word)
		count--
	}
	return words, nil
}

func QueryLLMAndInsertWords(words ...string) (map[string]*wordDesc, error, []string){
	errWords := make([]string,0)
	res := make(map[string]*wordDesc)
	wordDescs, err := GetWordDescFromLLM(words...)
	if err != nil{
		return nil, err, errWords
	}
	for word, wordDesc := range wordDescs{
		if wordDesc.Err == "true"{
			errWords = append(errWords, word)
			continue
		}
		err = insertWords(wordDesc)
		if err != nil{
			log.Println("insertWords error:", err)
			continue
		}
		err = esClient.IndexWordDesc(wordDesc)
		if err != nil{
			log.Println("esClient.IndexWordDesc error:", err)
			continue
		}
		err = redisClient.HSetWord(word, wordDesc.WordID)
		if err != nil{
			log.Println("redisClient.HSetWord error:", err)
			continue
		}
		res[word] = wordDesc
	}	
	return res, nil, errWords
}

// scaleUpWords 用 LLM 补全单词：pool 非空时用协程池执行，否则起临时 goroutine。
// 用户查词时传 LLMPool，批量导入等可传 nil 使用临时 worker。
func scaleUpWords(pool *GoRoutinePool, words ...string) (map[string]*wordDesc, error, []string) {
	errWords := make([]string, 0)
	if err := checkSyncLog(); err != nil {
		return nil, errors.New("sync log is not empty, please sync first"), errWords
	}
	res := make(map[string]*wordDesc)
	var mu sync.Mutex
	errSlice := make([]error, 0)
	var errMu sync.Mutex

	if pool != nil {
		// 使用全局协程池：按批提交任务，等待全部完成
		const batchSize = 10
		var wg sync.WaitGroup
		for i := 0; i < len(words); i += batchSize {
			end := i + batchSize
			if end > len(words) {
				end = len(words)
			}
			batch := make([]string, end-i)
			copy(batch, words[i:end])
			wg.Add(1)
			task := func(ctx context.Context) {
				defer wg.Done()
				portion, err, eWords := QueryLLMAndInsertWords(batch...)
				mu.Lock()
				for word, wd := range portion {
					res[word] = wd
				}
				mu.Unlock()
				if err != nil {
					errMu.Lock()
					errSlice = append(errSlice, err)
					errWords = append(errWords, eWords...)
					errMu.Unlock()
				}
			}
			if !pool.Submit(task) {
				wg.Done()
				return nil, errors.New("pool closed or busy"), errWords
			}
		}
		wg.Wait()
	} else {
		// 无池时：临时起 worker，逻辑与原实现一致
		goRoutineCount := 10
		wordsChan := make(chan string, len(words))
		var wg sync.WaitGroup
		for i := 0; i < goRoutineCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				assignment := make([]string, 0)
				for word := range wordsChan {
					assignment = append(assignment, word)
					if len(assignment) == 10 {
						portion, err, eWords := QueryLLMAndInsertWords(assignment...)
						if err != nil {
							errMu.Lock()
							errSlice = append(errSlice, err)
							errWords = append(errWords, eWords...)
							errMu.Unlock()
						}
						mu.Lock()
						for w, wd := range portion {
							res[w] = wd
						}
						mu.Unlock()
						assignment = make([]string, 0)
					}
				}
				if len(assignment) > 0 {
					portion, err, eWords := QueryLLMAndInsertWords(assignment...)
					if err != nil {
						errMu.Lock()
						errSlice = append(errSlice, err)
						errWords = append(errWords, eWords...)
						errMu.Unlock()
					}
					mu.Lock()
					for w, wd := range portion {
						res[w] = wd
					}
					mu.Unlock()
				}
			}()
		}
		for _, word := range words {
			wordsChan <- word
		}
		close(wordsChan)
		wg.Wait()
	}

	if len(errSlice) > 0 {
		for _, err := range errSlice {
			log.Println(err)
		}
		return nil, errors.New("scaleUpWords error"), errWords
	}
	return res, nil, errWords
}

func checkSyncLog() error {
	file, _ := os.OpenFile("log/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	logger := log.New(file, "", log.LstdFlags)
	logger.Println("checkSyncAndSave start")
	wordsInRedis, err := redisClient.HGetAllWords()
	if err != nil {
		logger.Println("redisWordClient.HGetAllWords error:", err)
		return err
	}
	rows, err := db.Query("SELECT id,word FROM vocabulary")
	if err != nil {
		logger.Println("db.Query error:", err)
		return err
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
		return err
	}
	wordsInEs, err := esClient.SearchAllWordIDs(500)
	if err != nil {
		logger.Println("words in es error:", err)
		return err
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
		return nil
	} else {
		logger.Println("Words are not synced, details are in missingWord")
	}
	esFile, err := os.OpenFile("log/missInEs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open es.log error:", err)
		return err
	}
	defer esFile.Close()
	mysqlFile, err := os.OpenFile("log/missInMysql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open mysql.log error:", err)
		return err

	}
	defer mysqlFile.Close()
	redisFile, err := os.OpenFile("log/missInRedis.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("open redis.log error:", err)
		return err
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
	return syncMissingFromLogs()
}

