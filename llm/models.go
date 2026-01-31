package llm

import (
	"context"
	"errors"
	_ "fmt"
	"log"
	"os"

	"google.golang.org/genai"

	// "github.com/openai/openai-go/v3"
	// "github.com/openai/openai-go/v3/option"
	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	volModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	_"github.com/openai/openai-go/v3"
)

func InitModels() error {
	gemini_api_key = os.Getenv("GEMINI_API_KEY")
	deepseek_api_key = os.Getenv("DEEPSEEK_API_KEY")
	ark_api_key = os.Getenv("ARK_API_KEY")
	dpModel, err := newAIModel(DEEP_SEEK)
	if err != nil {
		return err
	}
	GmModel, err := newAIModel(GEMINI)
	if err != nil {
		return err
	}
	ArkModel, err := newAIModel(ARK)
	if err != nil {
		return err
	}
	Models[DEEP_SEEK] = dpModel
	Models[GEMINI] = GmModel
	Models[ARK] = ArkModel
	return nil
}


func newAIModel(modelType ModelType) (AIModel, error) {
	switch modelType {
	case DEEP_SEEK:
		client, err := deepseek.NewClient(deepseek_api_key)
		if err != nil {
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
	QueryModel(string) (string, error)
	GetWordDefWithJson(string) (string, error)
	GetArticleWithJson([]string) (string, error)
}

type DeepseekModel struct {
	client  deepseek.Client
	api_key string
}
type GeminiModel struct {
	api_key string
	ctx     context.Context
	client  *genai.Client
}
type VolcanoModel struct {
	client *arkruntime.Client
	ctx    context.Context
}

func (ds *DeepseekModel) QueryModel(query string) (string, error) {
	chatReq := &request.ChatCompletionsRequest{
		Model:  deepseek.DEEPSEEK_CHAT_MODEL,
		Stream: false,
		Messages: []*request.Message{
			{
				Role:    "user",
				Content: query, // set your input message
			},
		},
	}
	chatResp, err := ds.client.CallChatCompletionsChat(context.Background(), chatReq)
	if err != nil {
		return "", err
	}
	return chatResp.Choices[0].Message.Content, nil
}


func (gemini *GeminiModel) QueryModel(query string) (string, error) {

	result, err := gemini.client.Models.GenerateContent(
		gemini.ctx,
		"gemini-2.5-flash",
		genai.Text(query),
		nil,
	)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return result.Text(), nil
}


func (vo *VolcanoModel) QueryModel(query string) (string, error) {
	req1 := volModel.CreateChatCompletionRequest{
		Model: "deepseek-v3-2-251201", //替换为Model ID，请从文档获取 https://www.volcengine.com/docs/82379/1330310
		Messages: []*volModel.ChatCompletionMessage{
			{
				Role: volModel.ChatMessageRoleUser,
				Content: &volModel.ChatCompletionMessageContent{
					StringValue: volcengine.String(query),
				},
			},
		},
	}

	resp1, err := vo.client.CreateChatCompletion(vo.ctx, req1)
	if err != nil {
		return "", err
	}
	return *resp1.Choices[0].Message.Content.StringValue, nil
}
