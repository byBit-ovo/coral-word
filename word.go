package main
import "fmt"

type Definition struct{
	Pos string 			`json:"pos"`
	Meanings []string   `json:"meaning"`
}
type Phrase struct{
	Example string 			`json:"example"`
	Example_cn []string 	`json:"example_cn"`
}

type wordDesc struct{
	Self string `json:"word"`
	Definitions []Definition `json:"definitions"`
	Derivatives []string `json:"derivatives"`
	Exam_tags   []string `json:"exam_tags"`
	Example 	string   `json:"example"`
	Example_cn 	string   `json:"example_cn"`
	Phrases  	[]Phrase `json:"phrases"`
	Synonyms    []string `json:"synonyms"`
}

func showWord(word *wordDesc){
	fmt.Println(word.Self)
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
		fmt.Println(phrase.Example)
		for _, cn := range phrase.Example_cn{
			fmt.Print(cn + " ")
		}
		fmt.Println()

	}
	fmt.Println(word.Synonyms)
}





