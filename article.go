package main

import "fmt"

type ArticleDesc struct {
	Err        string `json:"error"`
	Article    string `json:"article"`
	Article_cn string `json:"article_cn"`
}


func (atc *ArticleDesc) show(){
	fmt.Println("E.G.")
	fmt.Println(atc.Article)
	fmt.Println(atc.Article_cn)
	fmt.Println("--------------------------------------")
}