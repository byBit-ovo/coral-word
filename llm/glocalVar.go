package llm

var Models = map[int]AIModel{}

const (
	DEEP_SEEK = iota
	GEMINI
	ARK
)

var ModelsName = []string{
	"deepseek",
	"gemini",
	"ark",
}

const (
	WORD_QUERY = iota
	ARTICLE_QUERY
)

type ModelType int

var gemini_api_key string
var deepseek_api_key string
var ark_api_key string

var json_format_word = `{

"error":"false",
  "word": "expose",
  "pronunciation":"/ɪkˈspəʊz/"
  "definitions": [
{
  "pos": "vt.",
  "meaning": [
	"揭露,揭发",
	"使暴露",
	"使处于...作用(或影响)之下",
	"使面临",
	"(摄影)使曝光"
  ]
}
],
"derivatives": [
"exposed",
"exposes",
"exposure"
],
"exam_tags": [
"四级",
"六级",
"雅思",
"考研",
"专升本"
],
"example": "He threatened to expose the scandal to the public if they didn't pay him.",
"example_cn": "他威胁说，如果他们不付钱给他，他就向公众揭露这起丑闻。",
"phrases": [
{
	"example": "expose to",
	"example_cn": "使...暴露于"
},
{
	"example": "expose a secret",
	"example_cn": "揭露秘密"
}
],
"synonyms": [
"reveal",
"uncover",
"disclose",
"unmask"
]
}`
var json_format_article = `
{
	"error" : "xxx",
	"article" :"xxx",
	"article_cn" : "xxx"
}
`
var prompts = map[int]string{
	WORD_QUERY: "请以这样的json格式回复我(不要带任何多余符号,标点符号都用英文回复):" + json_format_word +
		",如果不存在这个单词,请将error设置为true,本次查询: ",
	ARTICLE_QUERY: `如果出错,请将error置为true,返回格式: ` + json_format_article +
		"请生成一篇包含下面几个单词的英语短文和中文翻译，以纯文本形式返回，不需要带任何多余符号，帮助用户记忆这些单词，" + "单词列表: ",
}
