# Runbook

## 本地环境约定

- `frontend` 使用 `npm`
- `ai-engine` 使用 `uv`
- `backend` 使用本机 `go`

默认端口：

- `frontend`: `5173`
- `backend`: `8080`
- `ai-engine`: `8000`

## 启动步骤

### 1. 启动 ai-engine

```bash
cd ai-engine
uv venv .venv
uv pip install --python .venv/bin/python -r requirements.txt
cp .env.example .env
uv run --python .venv/bin/python uvicorn app.main:app --host 127.0.0.1 --port 8000
```

可选配置：

- `LLM_DISABLED=true`
  - 强制走 Berry persona mock，适合第一次联调
- `LLM_PROVIDER=ollama`
  - 使用本地 Ollama
- `LLM_PROVIDER=deepseek`
  - 使用 DeepSeek API

推荐本地调试步骤：

```bash
cd ai-engine
cp .env.example .env
```

默认 `.env.example` 已设置为 `DeepSeek` 生成回复 + 本地 Ollama embedding。  
启动前需要在 `.env` 里填入 `DEEPSEEK_API_KEY`。  
如果想切回全本地 Ollama，可以用：

```bash
cp .env.ollama.example .env
```

说明：

- `LLM_TIMEOUT_SECONDS`
  - DeepSeek 模式默认 `60`
  - Ollama 模式建议 `180`，本地 Ollama 首次加载大模型可能较慢
- `DEEPSEEK_MODEL`
  - 默认 `deepseek-v4-flash`
  - 需要更强能力时可以改成 `deepseek-v4-pro`
- `DEEPSEEK_THINKING_ENABLED`
  - 默认 `false`
  - 当前聊天和 memory 抽取优先稳定、快速返回；需要更强推理时再开启
- 当前 RAG V1 默认检索仓库 `docs/`
  - 先走 Ollama 的 `qwen3-embedding:0.6b` 检索
  - 再用关键词结果做补充融合
  - embedding 模型不可用时会自动回退到关键词检索
  - 当前不依赖外部向量数据库

DeepSeek 模式下，生成回复不再依赖本地大语言模型，但 RAG embedding 仍然默认依赖本地 Ollama。先确保本地已经拉好 embedding 模型：

```bash
ollama pull qwen3-embedding:0.6b
```

如果使用 `.env.ollama.example` 切回全本地模式，再额外拉取生成模型：

```bash
ollama pull qwen3.5:4b
```

对应的 `.env` 默认配置是：

```env
RAG_EMBEDDING_PROVIDER=ollama
RAG_EMBEDDING_BASE_URL=http://127.0.0.1:11434
RAG_EMBEDDING_MODEL=qwen3-embedding:0.6b
RAG_EMBEDDING_TIMEOUT_SECONDS=60
```

### 2. 启动 backend

```bash
cd backend
go run .
```

可选环境变量：

- `PORT`
  - backend 监听端口，默认 `8080`
- `AI_ENGINE_BASE_URL`
  - ai-engine 地址，默认 `http://localhost:8000`
- `AI_ENGINE_TIMEOUT_SECONDS`
  - backend 调 ai-engine 的超时时间，默认 `120`
  - 本地 Ollama 首次加载模型较慢时，可以临时调大
- `MEMORY_STORE_PATH`
  - Memory V1 的本地持久化文件，默认 `.data/memory-store.json`
  - 保存消息日志、长期记忆和记忆来源信息

示例：

```bash
AI_ENGINE_BASE_URL=http://127.0.0.1:8000 AI_ENGINE_TIMEOUT_SECONDS=180 MEMORY_STORE_PATH=.data/memory-store.json go run .
```

### 3. 启动 frontend

```bash
cd frontend
npm install
npm run dev -- --host 127.0.0.1 --port 5173
```

前端开发服务器会把 `/api/*` 代理到 backend。

## 自动检查

### backend

```bash
cd backend
go test ./...
```

### frontend

```bash
cd frontend
npm run build
```

### ai-engine

```bash
cd ai-engine
./.venv/bin/python -m py_compile \
  app/main.py \
  app/schemas.py \
  app/persona/profile.py \
  app/persona/prompt_builder.py \
  app/persona/examples.py \
  app/persona/mock.py \
  app/llm/client.py \
  app/orchestration/generate_reply.py \
  app/retrieval/config.py \
  app/retrieval/embedder.py \
  app/retrieval/index_store.py \
  app/retrieval/schemas.py \
  app/retrieval/chunker.py \
  app/retrieval/loader.py \
  app/retrieval/retriever.py
./.venv/bin/python -m unittest discover -s tests -p 'test_*.py'
```

### Agent eval

默认关闭 DeepEval 遥测：

```bash
cd ai-engine
DEEPEVAL_TELEMETRY_OPT_OUT=1 ./.venv/bin/python -m evals.run_agent_eval
```

当前 eval 使用本地 fixture 和固定 Agent 任务集，先覆盖接口到页面生成链路。安装 `deepeval` 后，报告中的 `deepeval_available` 会变为 `true`。

报告包含以下本地质量维度和阈值：

- `task_completion >= 1.0`
- `tool_correctness >= 1.0`
- `plan_adherence >= 0.75`
- `plan_quality >= 0.75`
- `frontend_quality >= 0.8`
- `contextual_relevancy >= 0.75`
- `faithfulness >= 1.0`
- `persona_memory_safety >= 1.0`

## 手动联调

### 健康检查

```bash
curl http://127.0.0.1:8000/healthz
curl http://127.0.0.1:8080/healthz
```

浏览器访问：

```text
http://127.0.0.1:5173/health
```

### AI 服务直连

```bash
curl -X POST http://127.0.0.1:8000/generate \
  -H 'Content-Type: application/json' \
  -d '{"user_message":"hello"}'
```

DeepSeek 模式下，成功返回时 `context_used` 里会包含 `deepseek`；如果没有配置 `DEEPSEEK_API_KEY`，会回退到 `mock_fallback`。

如果 `docs/` 中命中相关内容，返回里的 `used_knowledge_chunk_ids` 会是非空数组。

常见的 `context_used` 组合：

- `persona`, `persona.examples`, `deepseek`
  - 表示使用 DeepSeek 生成回复
- `knowledge`, `knowledge.embedding`, `knowledge.keyword`
  - 表示用了 embedding + 关键词融合检索
- `knowledge`, `knowledge.keyword`, `knowledge.lexical_fallback`
  - 表示 embedding 不可用，回退到了关键词检索
- `memory`
  - 表示 backend 已经把长期记忆注入 ai-engine

### 后端主链路

```bash
curl -X POST http://127.0.0.1:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"hello"}'
```

### Memory V1

触发一次包含长期项目信息的对话：

```bash
curl -X POST http://127.0.0.1:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"我们这个项目的前端栈固定用 Vue 3 和 TypeScript，后端接口由 Go Gin 提供。"}'
```

查看已保存的长期记忆：

```bash
curl http://127.0.0.1:8080/api/memories
```

验证点：

- 响应里 `memory_candidate_count` 大于 `0` 时，说明 ai-engine 抽取到了候选记忆
- 响应里 `memory_written=true` 时，说明 backend 已经保存了高置信长期记忆
- 下一轮相关问题的 `context_used` 里出现 `memory` 时，说明 backend 已把长期记忆注入 ai-engine

### 前端联调

浏览器打开：

```text
http://127.0.0.1:5173/
```

验证点：

- 页面能正常打开
- 输入消息后能发送
- 页面显示 Berry 回复
- 页面显示 `trace_id`

## 常见问题

### `--port` requires an argument

命令被意外断行了。确保整条命令在一行，或者使用反斜杠换行：

```bash
uv run --python .venv/bin/python uvicorn app.main:app \
  --host 127.0.0.1 \
  --port 8000
```

### 前端页面能打开，但发送失败

优先检查：

1. backend 是否已启动
2. ai-engine 是否已启动
3. backend 的 `AI_ENGINE_BASE_URL` 是否指向正确地址

### `curl /api/chat` 失败

按顺序排查：

1. `curl http://127.0.0.1:8080/healthz`
2. `curl http://127.0.0.1:8000/healthz`
3. `curl -X POST http://127.0.0.1:8000/generate ...`
4. 再测 `POST /api/chat`

## 文档边界

这个文件只写：

- 怎么启动
- 怎么测试
- 怎么联调
- 怎么排查明显运行问题
