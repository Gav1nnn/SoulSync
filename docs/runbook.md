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
uv run --python .venv/bin/python uvicorn app.main:app --host 127.0.0.1 --port 8000
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
cp .env.example .env
# 编辑 .env，填入 DEEPSEEK_API_KEY（可选；未配置时 /generate 使用 mock）
uv pip install --python .venv/Scripts/python.exe -r requirements.txt
./.venv/Scripts/python.exe -m compileall app
```

环境变量（`ai-engine/.env`，见 `.env.example`）：

- `LLM_PROVIDER=ollama`：使用本地 Ollama（无需 API Key）
- `LLM_BASE_URL`：Ollama 默认 `http://127.0.0.1:11434/v1`
- `LLM_MODEL`：与 `ollama list` 中名称一致，如 `qwen2.5:3b`
- `LLM_DISABLED=true`：强制只用 mock
- 旧名 `DEEPSEEK_*` 仍兼容云端 DeepSeek

## Ollama 本地小模型接入

### 1. 安装 Ollama（Windows）

从 [https://ollama.com/download](https://ollama.com/download) 安装。安装后托盘程序会自动提供 API（默认 `11434` 端口）。

### 2. 拉取小模型

```powershell
ollama pull qwen2.5:3b
# 或其它：ollama pull llama3.2:3b
ollama list
```

### 3. 配置 SoulSync ai-engine

```powershell
cd ai-engine
copy .env.example .env
```

编辑 `.env`（方案 A）：

```env
LLM_PROVIDER=ollama
LLM_BASE_URL=http://127.0.0.1:11434/v1
LLM_MODEL=qwen2.5:3b
```

### 4. 启动顺序

```text
1. 确认 Ollama 在运行（托盘图标 / ollama list 能成功）
2. 启动 ai-engine（读取 .env）
3. 启动 backend
4. 启动 frontend
```

### 5. 验证

```powershell
# 应看到 llm_provider=ollama
curl http://127.0.0.1:8000/health

# 直连本地模型
curl -X POST http://127.0.0.1:8000/llm/deepseek/chat `
  -H "Content-Type: application/json" `
  -d '{"messages":[{"role":"user","content":"hello"}]}'

# 经 Go 主链路（前端聊天同此路径）
curl -X POST http://127.0.0.1:8080/api/chat `
  -H "Content-Type: application/json" `
  -d '{"message":"帮我拆一个列表页"}'
```

成功时回复为模型生成内容；`context_used` 含 `ollama`。Ollama 未启动或模型名错误时会回退 mock（`mock_fallback`）。

## 手动联调

### 健康检查

```bash
curl http://127.0.0.1:8000/health
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

### LLM 直连（Ollama / DeepSeek，需已配置 .env）

```bash
curl -X POST http://127.0.0.1:8000/llm/deepseek/chat \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"用一句话介绍 Vue3"}]}'
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
2. `curl http://127.0.0.1:8000/health`
3. `curl -X POST http://127.0.0.1:8000/generate ...`
4. 再测 `POST /api/chat`

## 文档边界

这个文件只写：

- 怎么启动
- 怎么测试
- 怎么联调
- 怎么排查明显运行问题
