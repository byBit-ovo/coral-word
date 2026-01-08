package main
import (
	"fmt"
	"github.com/byBit-ovo/coral_word/llm"
	"encoding/json"
)

type Definition struct{
	Pos string 			`json:"pos"`
	Meanings []string   `json:"meaning"`
}
type Phrase struct{
	Example string 			`json:"example"`
	Example_cn string 	`json:"example_cn"`
}
const(
	TagZsb = 1 << iota
	TagCET4
	TagCET6
	TagIELTS
	TagPostgrad
)




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
func TagsFromMask(mask int32) []string{
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
}


func QueryWord(word string) (*wordDesc, error){
	word_desc, err := selectWordByName(word)
	if err != nil{
		json_rsp, err := llm.Models[llm.DEEP_SEEK].GetDefinition(word)
		if err != nil{
			return nil, err
		}
		word_desc = &wordDesc{}
		err = json.Unmarshal([]byte(json_rsp), word_desc)
		if err != nil || word_desc.Err == "true"{
			return nil, err
		}
		err = insertWord(word_desc)
		if err != nil{
			return nil, err
		}
	}

	return word_desc, nil
}

func showWord(word *wordDesc){
	fmt.Println(word.Word, word.Pronunciation)
	for _, def := range word.Definitions{
		fmt.Println(def.Pos)
		for _, meaning := range def.Meanings{
			fmt.Print(meaning + " ")
		} 
		fmt.Println()
	}
	for _, der := range word.Derivatives{
		fmt.Print(der+" ")
	}
	fmt.Println()
	for _, tag := range word.Exam_tags{
		fmt.Print(tag + " ")
	}
	fmt.Println()
	fmt.Println(word.Example)
	fmt.Println(word.Example_cn)
	for _, phrase := range word.Phrases{
		fmt.Println(phrase.Example + " " + phrase.Example_cn)
	}
	fmt.Println(word.Synonyms)
}





