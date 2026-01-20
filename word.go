package main
import (
	"fmt"
	"github.com/byBit-ovo/coral_word/llm"
	_"encoding/json"
	"sort"
)

type Definition struct{
	Pos string 			`json:"pos"`
	Meanings []string   `json:"meaning"`
}
type Phrase struct{
	Example string 		`json:"example"`
	Example_cn string 	`json:"example_cn"`
}
// wordNote should be separated from wordDesc cus everyUser has their own note
type wordDesc struct{
	Err string `json:"error"`
	Word string `json:"word"`
	Pronunciation string `json:"pronunciation"`
	Definitions []Definition `json:"definitions"`
	Derivatives []string `json:"derivatives"`
	Exam_tags   []string `json:"exam_tags"`
	Example 	string   `json:"example"`
	Example_cn 	string   `json:"example_cn"`
	Phrases  	[]Phrase `json:"phrases"`
	Synonyms    []string `json:"synonyms"`
	Source 		int
	WordID 		int64
}
const(
	TagZsb = 1 << iota
	TagCET4
	TagCET6
	TagIELTS
	TagPostgrad
)

func insertWord(word *wordDesc)(err error){
	tags := aggregateTags(word.Exam_tags)
	tx, err := db.Begin()
	if err != nil {
    	return err
	}
	defer func() { 
		if err != nil{
			_ = tx.Rollback() 
		}
	}()
	res, err := tx.Exec(`insert into vocabulary (word, pronunciation, tag, source) values (?,?,?,?)`, word.Word, word.Pronunciation, tags,word.Source)
	if err != nil {
    	return err
	}
	word.WordID, err = res.LastInsertId()
	if err != nil{
		return err
	}
	for _, def := range word.Definitions{
		for _,tr := range def.Meanings{
			res, err = tx.Exec(`insert into vocabulary_cn (word_id, translation, pos) values (?,?,?)`,word.WordID,tr,def.Pos)
			if err != nil{
				return err
			}
		}

	}
	type wordPair struct{
		distance int
		word string
	}
	pairs := []wordPair{}
	for _,der := range(word.Derivatives){
		pairs = append(pairs, wordPair{minDistance(der, word.Word),der})
	}
	sort.Slice(pairs, func(i, j int) bool{
		if pairs[i].distance != pairs[j].distance{
			return pairs[i].distance < pairs[j].distance
		}
		return pairs[i].word < pairs[j].word
	})
	for i,pair := range pairs{
		if i>=3 {
			break
		}
		res, err = tx.Exec(`insert into derivatives (word_id, der) values (?,?)`,word.WordID, pair.word)
		if err != nil{
			return err
		}
	}
	for i, syn := range word.Synonyms{
		if i >= 3{
			break
		}
		res, err = tx.Exec("insert into synonyms (word_id, syn) values (?, ?)", word.WordID, syn)
		if err != nil{
			return err
		}

	}
	res, err = tx.Exec("insert into example (word_id, sentence, translation) values (?,?,?)", word.WordID, word.Example,word.Example_cn)
	if err != nil{
		return err
	}
	for i, phrase := range word.Phrases{
		if i >=5{
			break
		}
		res, err = tx.Exec("insert into phrases (word_id, phrase, translation) values (?,?,?)", word.WordID, phrase.Example,phrase.Example_cn)
		if err != nil{
			return err
		}
	}
	return tx.Commit()
}



func aggregateTags(tags []string) int32{
	count := 0
	for _, tag := range tags{
		switch tag {
		case "专升本": count += TagZsb
		case "四级": count += TagCET4
		case "六级": count += TagCET6
		case "雅思": count += TagIELTS
		case "考研": count += TagPostgrad
		}
	}
	return int32(count)
}
func TagsFromMask(mask int64) []string{
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



func QueryWord(word string) (*wordDesc, error){
	word_desc, err := selectWordByName(word)
	if err != nil{
		//query from llm
		word_desc, err = GetWordDesc(word)
		if err != nil{
			return nil, err
		}
		//insert into database
		err = insertWord(word_desc)
		return word_desc, err
	}
	return word_desc, nil
}

func (word *wordDesc)show(){
	fmt.Println("Source: ",llm.ModelsName[word.Source])
	fmt.Println(word.Word, word.Pronunciation)
	fmt.Print("TAG: ")
	for _, tag := range word.Exam_tags{
		fmt.Print(tag + " ")
	}
	fmt.Println()
	for _, def := range word.Definitions{
		fmt.Print(def.Pos," ")
		for _, meaning := range def.Meanings{
			fmt.Print(meaning + " ")
		} 
		fmt.Println()
	}
	fmt.Print("派生词汇: ")
	for _, der := range word.Derivatives{
		fmt.Print(der+" ")
	}
	fmt.Println()
	fmt.Println("E.G.", word.Example)
	fmt.Println("翻译: ",word.Example_cn)
	for _, phrase := range word.Phrases{
		fmt.Println(phrase.Example + " " + phrase.Example_cn)
	}
	fmt.Println("同义词: ", word.Synonyms)
	fmt.Println("-------------------------------------------------------------")
}

func (word *wordDesc) showExample(){
	fmt.Println(word.Example)
	fmt.Println(word.Example_cn)
}





