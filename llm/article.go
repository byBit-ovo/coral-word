package llm


func (vo *VolcanoModel) GetArticleWithJson(words []string) (string, error) {
	articleQuery := prompts[ARTICLE_QUERY]
	for _, words := range words {
		articleQuery += (words + " ")
	}
	return vo.QueryModel(articleQuery)
}

func (gemini *GeminiModel) GetArticleWithJson(words []string) (string, error) {
	articleQuery := prompts[ARTICLE_QUERY]
	for _, words := range words {
		articleQuery += (words + " ")
	}
	return gemini.QueryModel(articleQuery)
}

func (ds *DeepseekModel) GetArticleWithJson(words []string) (string, error) {
	articleQuery := prompts[ARTICLE_QUERY]
	for _, words := range words {
		articleQuery += (words + " ")
	}
	return ds.QueryModel(articleQuery)
}
