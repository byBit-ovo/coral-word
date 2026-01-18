package llm


func (ds *DeepseekModel) GetWordDefWithJson(word string) (string, error) {
	return ds.QueryModel(prompts[WORD_QUERY] + word)
}

func (gemini *GeminiModel) GetWordDefWithJson(word string) (string, error) {
	return gemini.QueryModel(prompts[WORD_QUERY] + word)
}

func (vo *VolcanoModel) GetWordDefWithJson(word string) (string, error) {
	return vo.QueryModel(prompts[WORD_QUERY] + word)
}