package llm

import(
	"strings"
)



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

const json_format_word = `你是一个词汇查询 API,请根据给定单词返回 JSON 数据，格式如下：

{
  "error":"false",
  "word":"<单词>",
  "pronunciation":"<音标>",
  "definitions":[{"pos":"<词性>","meaning":["<中文意思1>", "..."]}],
  "derivatives":["<派生词1>", "..."],
  "exam_tags":["四级","六级","雅思","考研","专升本"],
  "example":"<英文例句>",
  "example_cn":"<中文翻译>",
  "phrases":[{"example":"<短语>","example_cn":"<中文翻译>"}],
  "synonyms":["<同义词1>", "..."]
}

要求：
- 不输出解释、文字说明、Markdown 或转义字符
- 保持 JSON 可解析
- 每个数组中的数据不要超过10个
- 对每个字段填充合理内容，如果没有，填空数组
- 例句自然，短语真实
- 如果单词不存在,或者不是常见或者常用的单词,请将error设置为true,其他字段填空即可
这次查询的单词: `

const json_format_article = 
`你是一个英语写作和词汇助手。用户会提供一个英语单词列表。请根据这些单词生成一篇自然流畅的英语文章，文章长度在 300 到 800 词之间，并在文章中合理地使用这些单词。然后生成中文释义版本的文章。

输出必须是 JSON 格式，如下：

{
    "error": "false",
    "article": "<完整英文文章，包含所有单词>",
    "article_cn": "<对应中文翻译文章>"
}

要求：
1. JSON 必须严格可解析，不要输出额外文字或解释。
2. article 要自然连贯，不要像列表或者堆砌单词。
3. article_cn 要尽量忠实于英文文章，通顺易读。
4. 如果文章无法生成,error 字段置为 "true"，并用空字符串填充 article 和 article_cn。
5. 保证每个单词至少在英文文章中出现一次。
这次查询的单词列表:
`
var prompts = map[int]string{
	WORD_QUERY: json_format_word,
	ARTICLE_QUERY: json_format_article,
}

func GetWordPrompt(word string) string{
	return prompts[WORD_QUERY] + word
} 

func GetArticlePrompt(words []string) string{
	return prompts[ARTICLE_QUERY] + strings.Join(words, ",")
}
