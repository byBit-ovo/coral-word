# study-agent

一个针对 `GoLearning` 项目的学习助手，支持：

- 单词复习（简化 SM-2 间隔重复）
- 阅读推荐（基于主题和难度）
- 助手对话入口（复习/推荐意图）

## Run

```bash
cd /home/qzr/gitee/GoLearning/study-agent
go run .
```

默认监听：`http://localhost:9030`

可选环境变量：

- `PORT`：服务端口（默认 `9030`）
- `STUDY_AGENT_DATA`：数据文件路径（默认 `./data/study_data.json`）

## API

### 1) 添加或更新单词

`POST /words`

```json
{
  "term": "goroutine",
  "meaning": "go 协程",
  "example": "Use goroutine for concurrent tasks",
  "tags": ["concurrency", "go"]
}
```

### 2) 获取待复习单词

`GET /words/due?limit=10`

### 3) 提交复习结果

`POST /review`

```json
{
  "term": "goroutine",
  "score": 4
}
```

`score` 范围：`0-5`

### 4) 获取阅读推荐

`POST /reading/recommend`

```json
{
  "topics": ["http", "go-basics"],
  "current_level": 2,
  "limit": 3
}
```

### 5) 助手对话入口

`POST /assistant/chat`

```json
{
  "message": "给我推荐阅读"
}
```

## 示例命令

```bash
curl -s http://localhost:9030/health

curl -s -X POST http://localhost:9030/words \
  -H 'Content-Type: application/json' \
  -d '{"term":"goroutine","meaning":"go 协程","example":"Use goroutine","tags":["go","concurrency"]}'

curl -s http://localhost:9030/words/due?limit=5

curl -s -X POST http://localhost:9030/review \
  -H 'Content-Type: application/json' \
  -d '{"term":"goroutine","score":5}'

curl -s -X POST http://localhost:9030/reading/recommend \
  -H 'Content-Type: application/json' \
  -d '{"topics":["rag","http"],"current_level":3,"limit":3}'
```
