package llm


func (ds *DeepseekModel) GetWordDefWithJson(word ...string) (string, error) {
	return ds.QueryModel(GetWordPrompt(word...))
}

func (gemini *GeminiModel) GetWordDefWithJson(word... string) (string, error) {
	return gemini.QueryModel(GetWordPrompt(word...))
}

func (vo *VolcanoModel) GetWordDefWithJson(word... string) (string, error) {
	return vo.QueryModel(GetWordPrompt(word...))
}