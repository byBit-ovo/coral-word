# Python LLM 服务设计

## 架构

```
[Go coral_word]  --gRPC-->  [Python LLM 服务]  --API-->  [DeepSeek / Gemini / 火山等]
```

- **Go**：查词、业务逻辑、MySQL/Redis/ES；需要补全生词时通过 **gRPC** 调用 Python 服务。
- **Python**：组 prompt、调大模型、解析 JSON，通过 gRPC 返回与 proto `WordDesc` 一致的结果。

## 环境变量

| 端 | 变量 | 说明 |
|----|------|------|
| Go | `LLM_GRPC_TARGET` | gRPC 地址（如 `localhost:50052`），优先使用 |
| Go | `LLM_SERVICE_URL` | HTTP 回退地址（如 `http://localhost:8000`），未设 gRPC 时使用 |
| Python | `DEEPSEEK_API_KEY` / `GEMINI_API_KEY` / `OPENAI_API_KEY` | 大模型 API Key |
| Python | `LLM_PROVIDER` | 可选：`deepseek` / `gemini` / `openai`，默认 `deepseek` |
| Python | `PORT` | gRPC 监听端口，默认 **50052** |

## API 约定（gRPC）

proto 见 `proto/coral_word.proto`，服务 `LLMService`：

- **WordDefinitions(WordDefinitionsRequest) returns (WordDefinitionsResponse)**  
  - 请求：`repeated string words`  
  - 响应：`repeated WordDesc words`（与现有 `WordDesc` 定义一致）

- **Article(ArticleRequest) returns (ArticleResponse)**  
  - 请求：`repeated string words`  
  - 响应：`error`, `article`, `article_cn`

Go 端若设置 `LLM_GRPC_TARGET`，则用 `pb.NewLLMServiceClient(conn).WordDefinitions` / `Article` 调用，并将 `pb.WordDesc` 转为内部 `wordDesc`（`FromPbWordDesc`）。未设 gRPC 时仍可回退到 `LLM_SERVICE_URL` 的 HTTP 或直连 Go LLM。

## Python 服务目录（已实现）

```
coral_word/llm_service/
├── __init__.py
├── main.py          # FastAPI app，/word_definitions、/article、/health
├── schemas.py       # Request/Response Pydantic 模型
├── prompts.py      # 与 Go LLM/prompts.go 一致的 prompt 文本
├── llm_client.py   # 调 DeepSeek/OpenAI/Gemini
├── requirements.txt
└── README.md
```

## 快速开始

1. 生成 Python gRPC 代码（在 coral_word 根目录，首次或 proto 变更后）：
   ```bash
   pip install -r llm_service/requirements.txt
   python -m grpc_tools.protoc -I proto --python_out=. --grpc_python_out=. proto/coral_word.proto
   ```
2. 启动 Python gRPC 服务（在 coral_word 根目录）：
   ```bash
   export DEEPSEEK_API_KEY=your_key
   python -m llm_service.server_grpc
   ```
   默认监听 `[::]:50052`。
3. 在 Go 端 `.env` 中设置 `LLM_GRPC_TARGET=localhost:50052`，启动 coral_word 即可通过 gRPC 走 Python 服务补全生词。
