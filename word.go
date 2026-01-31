package main

import (
	_ "encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/byBit-ovo/coral_word/llm"
)

type Definition struct {
	Pos      string   `json:"pos"`
	Meanings []string `json:"meaning"`
}
type Phrase struct {
	Example    string `json:"example"`
	Example_cn string `json:"example_cn"`
}

// wordNote should be separated from wordDesc cus everyUser has their own note
type wordDesc struct {
	Err           string       `json:"error"`
	Word          string       `json:"word"`
	Pronunciation string       `json:"pronunciation"`
	Definitions   []Definition `json:"definitions"`
	Derivatives   []string     `json:"derivatives"`
	Exam_tags     []string     `json:"exam_tags"`
	Example       string       `json:"example"`
	Example_cn    string       `json:"example_cn"`
	Phrases       []Phrase     `json:"phrases"`
	Synonyms      []string     `json:"synonyms"`
	Source        int
	WordID        int64
	SelectedNotes map[string]string 
}

const (
	TagZsb = 1 << iota
	TagCET4
	TagCET6
	TagIELTS
	TagPostgrad
)

func insertWords(words ...*wordDesc) (err error) {
	if len(words) == 0 {
		return nil
	}
	vocs := make([]interface{}, 0, 4*len(words))
	placeholderGroups := make([]string, len(words))
	for i, w := range words {
		vocs = append(vocs, w.Word, w.Pronunciation, aggregateTags(w.Exam_tags), w.Source)
		placeholderGroups[i] = "(?, ?, ?, ?)"
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	placeholders := strings.Join(placeholderGroups, ",")
	res, err := tx.Exec("insert into vocabulary (word, pronunciation, tag, source) values "+placeholders, vocs...)
	if err != nil {
		return err
	}

	firstID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for i, w := range words {
		w.WordID = firstID + int64(i)

		for _, def := range w.Definitions {
			for _, tr := range def.Meanings {
				_, err = tx.Exec(`insert into vocabulary_cn (word_id, translation, pos) values (?,?,?)`, w.WordID, tr, def.Pos)
				if err != nil {
					return err
				}
			}
		}

		type wordPair struct {
			distance int
			word     string
		}
		pairs := []wordPair{}
		for _, der := range w.Derivatives {
			pairs = append(pairs, wordPair{minDistance(der, w.Word), der})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].distance != pairs[j].distance {
				return pairs[i].distance < pairs[j].distance
			}
			return pairs[i].word < pairs[j].word
		})
		for k, pair := range pairs {
			if k >= 3 {
				break
			}
			_, err = tx.Exec(`insert into derivatives (word_id, der) values (?,?)`, w.WordID, pair.word)
			if err != nil {
				return err
			}
		}

		for k, syn := range w.Synonyms {
			if k >= 3 {
				break
			}
			_, err = tx.Exec("insert into synonyms (word_id, syn) values (?, ?)", w.WordID, syn)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec("insert into example (word_id, sentence, translation) values (?,?,?)", w.WordID, w.Example, w.Example_cn)
		if err != nil {
			return err
		}

		for k, phrase := range w.Phrases {
			if k >= 5 {
				break
			}
			_, err = tx.Exec("insert into phrases (word_id, phrase, translation) values (?,?,?)", w.WordID, phrase.Example, phrase.Example_cn)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func aggregateTags(tags []string) int32 {
	count := 0
	for _, tag := range tags {
		switch tag {
		case "专升本":
			count += TagZsb
		case "四级":
			count += TagCET4
		case "六级":
			count += TagCET6
		case "雅思":
			count += TagIELTS
		case "考研":
			count += TagPostgrad
		}
	}
	return int32(count)
}
func TagsFromMask(mask int64) []string {
	tags := []string{}
	if mask&TagZsb != 0 {
		tags = append(tags, "专升本")
	}
	if mask&TagCET4 != 0 {
		tags = append(tags, "四级")
	}
	if mask&TagCET6 != 0 {
		tags = append(tags, "六级")
	}
	if mask&TagIELTS != 0 {
		tags = append(tags, "雅思")
	}
	if mask&TagPostgrad != 0 {
		tags = append(tags, "考研")
	}
	return tags
}

func QueryWords(word ...string) (map[string]*wordDesc, error, []string) {
	wordsInMysql := make([]string, 0)
	wordsToQuery := make([]string, 0)
	for _, w := range word {
		if _, err := redisClient.HGetWord(w); err != nil {
			wordsToQuery = append(wordsToQuery, w)
		} else {
			wordsInMysql = append(wordsInMysql, w)
		}
	}
	res := make(map[string]*wordDesc)
	var err error
	if len(wordsInMysql) > 0 {
		res, err = selectWordsByNames(wordsInMysql...)
		if err != nil {
			return nil, err, nil
		}
	}
	errWords := make([]string, 0)
	//query from llm
	if len(wordsToQuery) > 0 {
		for _, w := range wordsToQuery {
			wd, err := GetWordDesc(w)
			if err != nil {
				errWords = append(errWords, w)
				continue
			}
			res[w] = wd
			//insert into database
			err = insertWords(wd)
			if err != nil {
				log.Fatal("insertWord error:", err)
			}
			//insert into es
			err = esClient.IndexWordDesc(wd)
			if err != nil {
				log.Fatal("esClient.IndexWordDesc error:", err)
			}
			//insert into redis
			if err = redisClient.HSetWord(w, wd.WordID); err != nil {
				log.Fatal("redisWordClient.HSetWord error:", err)
			}

		}
		log.Println("Total word count from llm:", len(wordsToQuery)-len(errWords))
		return res, err, errWords
	}
	return res, nil, errWords
}

func (word *wordDesc) show() {
	fmt.Println("Source: ", llm.ModelsName[word.Source])
	fmt.Println(word.Word, word.Pronunciation)
	fmt.Print("TAG: ")
	for _, tag := range word.Exam_tags {
		fmt.Print(tag + " ")
	}
	fmt.Println()
	for _, def := range word.Definitions {
		fmt.Print(def.Pos, " ")
		for _, meaning := range def.Meanings {
			fmt.Print(meaning + " ")
		}
		fmt.Println()
	}
	fmt.Print("派生词汇: ")
	for _, der := range word.Derivatives {
		fmt.Print(der + " ")
	}
	fmt.Println()
	fmt.Println("E.G.", word.Example)
	fmt.Println("翻译: ", word.Example_cn)
	for _, phrase := range word.Phrases {
		fmt.Println(phrase.Example + " " + phrase.Example_cn)
	}
	fmt.Println("同义词: ", word.Synonyms)
	fmt.Println("精选用户笔记:***************************************************")
	for userName, note := range word.SelectedNotes {
		fmt.Println("用户: ", userName)
		fmt.Println("笔记: ", note)
	}
	fmt.Println("-------------------------------------------------------------")
}

func (word *wordDesc) showExample() {
	fmt.Println(word.Example)
	fmt.Println(word.Example_cn)
}


func eliminateWords(words []string) error {
	err := deleteWordsFromMysql(words...)
	if err != nil {
		return err
	}
	for _, w := range words{
		err = esClient.DeleteWordByName(w)
		if err != nil {
			return err
		}
		err = redisClient.HDelWord(w)
		if err != nil {
			return err
		}
	}
	return nil
}
