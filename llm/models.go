package llm

import(
	"os"
	_"fmt"
	"log"
	"context"
	"errors"
    "google.golang.org/genai"
	// "github.com/openai/openai-go/v3"
	// "github.com/openai/openai-go/v3/option"
	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	volModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)
func InitModels()error{
	gemini_api_key = os.Getenv("GEMINI_API_KEY")
	deepseek_api_key = os.Getenv("DEEPSEEK_API_KEY")
	ark_api_key = os.Getenv("ARK_API_KEY")
	dpModel, err := newAIModel(DEEP_SEEK)
	if err != nil{
		return err
	}
	GmModel, err := newAIModel(GEMINI)
	if err != nil{
		return err
	}
	ArkModel, err := newAIModel(ARK)
	if err != nil{
		return err
	}
	Models[DEEP_SEEK] = dpModel
	Models[GEMINI] = GmModel
	Models[ARK] = ArkModel
	return nil
}
var Models = map[int]AIModel{}
const (
	DEEP_SEEK = iota
	GEMINI
	ARK
)
const (
	WORD_QUERY = iota
)
type ModelType int
var gemini_api_key string
var deepseek_api_key string
var ark_api_key string
var json_format = `{
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
var prompts = map[int]string{
	WORD_QUERY: "请以这样的json格式回复我(不要带任何多余符号,标点符号都用英文回复):" + json_format +
	"列举出该单词的所有词性以及对应的中文释义、发音、10个左右数量的该单词的派生词(派生词" + 
	"定义为和该单词有几分相似的单词)、该单词是否是四级、六级、雅思、考研、专升本的核心词汇、" +
	"一条包含该单词的典型例句、" + 
	"包含该单词的几个典型短语(如果有)、该单词的近义词。本次查询:",
}




func newAIModel(modelType ModelType) (AIModel, error){
	switch modelType{
	case DEEP_SEEK:
		client, err := deepseek.NewClient(deepseek_api_key)
		if err != nil{
			return nil, err
		}
		return &DeepseekModel{client, deepseek_api_key}, nil
	case GEMINI:
		ctx := context.Background()
		// The client gets the API key from the environment variable `GEMINI_API_KEY`.
		client, err := genai.NewClient(ctx, nil)
		if err != nil {
			log.Fatal(err)
		}
		return &GeminiModel{gemini_api_key, ctx, client}, nil
	case ARK:
		ctx := context.Background()
		client := arkruntime.NewClientWithApiKey(ark_api_key)
		return &VolcanoModel{client, ctx}, nil
	}
	return nil, errors.New("Model not found")
}
// AIModel defines the interface for querying word definitions
type AIModel interface {
    GetDefinition(string) (string, error)
}
type DeepseekModel struct{
	client deepseek.Client
	api_key string
}
type GeminiModel struct{
	api_key string
	ctx context.Context
	client *genai.Client
}
type VolcanoModel struct{
	client *arkruntime.Client
	ctx context.Context

}
func (ds *DeepseekModel) GetDefinition(word string) (string, error){
	chatReq := &request.ChatCompletionsRequest{
		Model:  deepseek.DEEPSEEK_CHAT_MODEL,
		Stream: false,
		Messages: []*request.Message{
			{
				Role:    "user",
				Content: prompts[WORD_QUERY] + word, // set your input message
			},
		},
	}
	chatResp, err := ds.client.CallChatCompletionsChat(context.Background(), chatReq)
	if err != nil {
		return "", err
	}
	return chatResp.Choices[0].Message.Content, nil
}
func (gemini *GeminiModel)GetDefinition(word string) (string, error){

    result, err := gemini.client.Models.GenerateContent(
        gemini.ctx,
        "gemini-2.5-flash",
        genai.Text(prompts[WORD_QUERY] + word),
        nil,
    )
    if err != nil {
        log.Fatal(err)
		return "", err
    }
    return result.Text(), nil
}

func (model *VolcanoModel)GetDefinition(word string) (string, error){
	req1 := volModel.CreateChatCompletionRequest{
       Model: "doubao-seed-1-6-lite-251015",  //替换为Model ID，请从文档获取 https://www.volcengine.com/docs/82379/1330310
       Messages: []*volModel.ChatCompletionMessage{
          {
             Role: volModel.ChatMessageRoleUser,
             Content: &volModel.ChatCompletionMessageContent{
                StringValue: volcengine.String(prompts[WORD_QUERY] + "set"),
             },
          },
       },
    }

    resp1, err := model.client.CreateChatCompletion(model.ctx, req1)
    if err != nil {
		log.Fatal(err)
       	return "", err
    }
	return *resp1.Choices[0].Message.Content.StringValue, nil
}

// client := openai.NewClient(
// 	option.WithAPIKey(Gemini_api_key), // defaults to os.LookupEnv("OPENAI_API_KEY")
// )
// chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
// 	Messages: []openai.ChatCompletionMessageParamUnion{
// 		openai.UserMessage("今天吃啥"),
// 	},
// 	Model: openai.ChatModelGPT5,
// })
// if err != nil {
// 	panic(err.Error())
// }
// println(chatCompletion.Choices[0].Message.Content)


// var doubao_seed_1_8_251215 = "doubao-seed-1-8-251215"
// var doubao_seed_code_preview_251028 = "doubao-seed-code-preview-251028"
// var doubao_seed_1_6_lite_251015 = "doubao-seed-1-6-lite-251015"
// var doubao_seed_1_6_flash_250828 = "doubao-seed-1-6-flash-250828"
// var doubao_seed_1_6_vision_250815 = "doubao-seed-1-6-vision-250815"

// func Volai(){
// 	client := arkruntime.NewClientWithApiKey(Ark_api_key)
//     ctx := context.Background()
//     // 第一次请求
//     req1 := volModel.CreateChatCompletionRequest{
//        Model: "doubao-seed-1-6-lite-251015",  //替换为Model ID，请从文档获取 https://www.volcengine.com/docs/82379/1330310
//        Messages: []*volModel.ChatCompletionMessage{
//           {
//              Role: volModel.ChatMessageRoleUser,
//              Content: &volModel.ChatCompletionMessageContent{
//                 StringValue: volcengine.String(prompts[WORD_QUERY] + "set"),
//              },
//           },
//        },
//     }

//     resp1, err := client.CreateChatCompletion(ctx, req1)
//     if err != nil {
// 		log.Fatal(err)
//        	return
//     }
//     fmt.Println(*resp1.Choices[0].Message.Content.StringValue)
// }

