# Coral Word LLM Service (Python gRPC)

与 Go coral_word 通过 **gRPC** 对接的 Python 大模型服务：提供单词释义补全、文章生成，内部调用 DeepSeek / OpenAI / Gemini。

## 环境变量

| 变量 | 说明 |
|------|------|
| `DEEPSEEK_API_KEY` 或 `OPENAI_API_KEY` | 必填，大模型 API Key |
| `GEMINI_API_KEY` | 使用 Gemini 时必填 |
| `LLM_PROVIDER` | 可选：`deepseek` / `openai` / `gemini`，默认 `deepseek` |
| `PORT` | 可选，gRPC 监听端口，默认 **50052** |

## 生成 Python gRPC 代码（首次或 proto 变更后）

在 **coral_word 项目根目录** 执行：

```bash
pip install -r llm_service/requirements.txt
python -m grpc_tools.protoc -I proto --python_out=. --grpc_python_out=. proto/coral_word.proto
```

会在根目录生成 `coral_word_pb2.py`、`coral_word_pb2_grpc.py`。

## 启动 gRPC 服务

在 **coral_word 根目录** 执行：

```bash
export DEEPSEEK_API_KEY=your_key
python -m llm_service.server_grpc
```

默认监听 `[::]:50052`。修改端口：`PORT=50053 python -m llm_service.server_grpc`。

## Go 端使用

在 coral_word 的 `.env` 中设置：

```
LLM_GRPC_TARGET=localhost:50052
```

Go 会在需要补全时通过 gRPC 调用 Python 服务，不再直连大模型。

- **gRPC 优先**：若设置了 `LLM_GRPC_TARGET`，走 gRPC。
- **HTTP 回退**：若未设置 gRPC 但设置了 `LLM_SERVICE_URL`，仍走 HTTP。
- **直连 LLM**：两者都未设置则使用 Go 内置 LLM 调用。
