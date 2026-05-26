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

默认 `.env.example` 已设置为 `Ollama` 本地调用。  
如果本地没有启动 Ollama，ai-engine 会回退到 `mock fallback`。  
上线时再把 `.env` 改成 `DeepSeek` 配置即可。

说明：

- `LLM_TIMEOUT_SECONDS`
  - 当前默认 `180`
  - 本地 Ollama 首次加载大模型可能较慢，超时后会自动回退到 `mock fallback`

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

示例：

```bash
AI_ENGINE_BASE_URL=http://127.0.0.1:8000 go run .
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
  app/orchestration/generate_reply.py
./.venv/bin/python -m unittest discover -s tests -p 'test_*.py'
```

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

### 后端主链路

```bash
curl -X POST http://127.0.0.1:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"hello"}'
```

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
