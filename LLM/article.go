package llm


func (vo *VolcanoModel) GetArticleWithJson(words []string) (string, error) {
	
	return vo.QueryModel(GetArticlePrompt(words))
}

func (gemini *GeminiModel) GetArticleWithJson(words []string) (string, error) {
	return gemini.QueryModel(GetArticlePrompt(words))
}

func (ds *DeepseekModel) GetArticleWithJson(words []string) (string, error) {
	return ds.QueryModel(GetArticlePrompt(words))
}
