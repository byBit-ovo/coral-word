# Coral Word — 英语单词学习与复习系统

## 项目简介

基于 Go 的英语单词查询与复习后端服务，支持多数据源（MySQL / Redis / Elasticsearch）读写分离与缓存、LLM 补全生词、SM-2 间隔复习算法，提供 HTTP（Gin）与 gRPC 双协议接口，可作为简历中的**全栈型后端项目**展示。

---

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 / 运行时 | Go 1.25 |
| Web 框架 | Gin |
| 数据存储 | MySQL、Redis、Elasticsearch |
| 大模型 | DeepSeek / Gemini / 火山方舟（多模型可选，用于生词释义补全） |
| 服务发现 | etcd（gRPC 服务注册） |
| 并发控制 | 协程池、singleflight 防缓存击穿 |

---

## 核心功能与实现要点

### 1. 单词查询（读多写少 + 缓存分层）

- **读路径**：Redis（word → wordId）→ MySQL（wordId → 单词详情）；未命中时走 Elasticsearch 模糊检索，再回写 Redis / MySQL。
- **写路径**：生词通过协程池异步调用 LLM 补全释义，写入 MySQL、ES、Redis，并记录缺失日志便于离线补偿同步。
- **防击穿**：同一单词并发请求使用 `singleflight.Group` 合并为单次 LLM/DB 调用，结果共享。

### 2. 用户与笔记本

- 用户注册 / 登录；Session 存 Redis（sessionId → userId），支持多端。
- 笔记本：按用户 + 笔记本名维度的单词本，生词加入笔记本、创建笔记本等，为复习提供数据源。

### 3. 间隔复习（SM-2 简化版）

- 基于 `learning_record`（熟悉度、连续正确次数、下次复习时间）拉取待复习单词，生成多轮次队列（熟悉度低则出现次数多）。
- HTTP 接口：`/review/start`（按 session + 笔记本名创建复习会话）→ `/review/next`（拉下一题）→ `/review/submit`（提交认识/不认识，更新熟悉度与下次复习时间）；复习结束后事务写回 MySQL（学习记录 + 用户 streak）。

### 4. 笔记 CRUD

- 单词笔记的创建、更新、查询、删除，与用户和单词绑定，支撑「查词 + 记笔记 + 复习」闭环。

### 5. 多协议与部署

- **HTTP**：Gin 提供 RESTful API（登录、查词、笔记本、复习、笔记等）。
- **gRPC**：单词查询等 RPC 接口，可配合 etcd 做服务注册与发现。
- 支持仅 HTTP、仅 gRPC 或双服务同时启动，通过环境变量配置。

---

## 项目亮点（简历可写）

- **多存储协同**：Redis 缓存 + MySQL 主数据 + Elasticsearch 检索，读路径分层、写路径异步补全与补偿同步。
- **高并发与稳定性**：协程池限制 LLM 并发、singleflight 防重复请求、复习会话内存锁 + 会话级状态机。
- **LLM 集成**：多模型封装（DeepSeek / Gemini / 火山方舟），生词释义自动补全并落库。
- **复习算法**：SM-2 简化实现（熟悉度 0–5、间隔天数、随机抖动），多轮次队列与事务持久化。
- **工程化**：gRPC + etcd、环境变量与 .env 配置、结构化 API 响应（code/message/data）。

---

## 本地运行

```bash
# 依赖：MySQL、Redis、Elasticsearch、.env 中配置的 DB/Redis/ES 及 LLM API Key

# 仅 HTTP（Gin）
HTTP_ADDR=0.0.0.0:8080 go run .

# 仅 gRPC
GRPC_ADDR=0.0.0.0:50051 go run .

# HTTP + gRPC
GRPC_ADDR=0.0.0.0:50051 HTTP_ADDR=0.0.0.0:8080 go run .
```

**etcd（gRPC 服务注册）**：配置 `ETCD_ENDPOINTS`、`ETCD_SERVICE_NAME` 后，gRPC 地址会注册到 etcd。

**Nginx 反向代理**：若需对外 80 端口或配合域名部署，见 [nginx/README.md](nginx/README.md)。

---

## 简历一句话示例

**Coral Word（Go 后端）**  
英语单词查询与复习系统：MySQL/Redis/ES 多源读写与缓存、LLM 生词补全、SM-2 间隔复习；Gin REST + gRPC 双协议，协程池与 singleflight 控制并发与缓存击穿；支持笔记本、笔记与复习会话管理。
