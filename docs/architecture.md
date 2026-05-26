# Architecture

## 项目定位

SoulSync 是一个面向后端开发者的 AI 协作项目。

产品定位不是普通聊天机器人，也不是多角色社区，而是一个带稳定人设的前端协作助手：`Berry`。

Berry 的核心价值：

- 帮后端开发者补前端能力
- 协助页面生成、组件编写、接口联调、前端 Debug
- 以稳定 persona 输出建议，而不是随机聊天
- 具备长期记忆的功能

## 核心原则

> Go 管业务，Python 管 AI。

这条边界是当前架构最重要的约束：

- `frontend` 负责交互展示
- `backend` 负责业务接口、主流程控制、后续持久化
- `ai-engine` 负责 AI 相关能力，如 Persona、Prompt、RAG、Memory 抽取、模型调用

## 当前调用链路

```text
用户输入
-> frontend
-> POST /api/chat
-> backend
-> POST /generate
-> ai-engine
-> backend
-> frontend 展示回复
```

当前 backend 同时负责：

- 校验输入
- 请求 ai-engine
- 生成 `trace_id`
- 记录最小 trace
- 把统一响应返回给 frontend

当前 ai-engine 在 Persona V1 中负责：

- 接收结构化 Berry persona
- 组装 system instruction
- 注入 Berry few-shot 示例
- 按配置调用 Ollama / DeepSeek
- 在模型不可用时回退 persona mock

## 当前目录职责

### frontend

- 技术：`Vue 3 + TypeScript + Vite`
- 当前页面：
  - `/`
    - 最小聊天页
  - `/health`
    - 前端健康页
- 当前职责：
  - 收集用户输入
  - 请求 `/api/chat`
  - 展示用户消息、Berry 回复、错误态、加载态、`trace_id`

### backend

- 技术：`Go + Gin`
- 当前职责拆分：
  - `main.go`
    - 应用入口，组装依赖
  - `internal/app/http.go`
    - 路由和 handler
  - `internal/app/service.go`
    - 聊天主流程
  - `internal/app/ai_client.go`
    - 调用 ai-engine
  - `internal/app/trace_store.go`
    - 内存态 trace 存储
  - `internal/app/types.go`
    - 请求、响应、trace 类型

### ai-engine

- 技术：`Python + FastAPI`
- 当前职责：
  - 提供健康检查
  - 接收结构化生成请求
  - 维护 Berry 的结构化 persona
  - 拼装 persona prompt 和 few-shot 示例
  - 调用模型或回退 mock
  - 返回 `context_used`

## 当前接口边界

### frontend -> backend

`POST /api/chat`

请求：

```json
{ "message": "hello" }
```

响应：

```json
{
  "reply": "我是 Berry。你这句“hello”我已经接住了。先别急着乱堆页面，我会先帮你把结构、状态和接口边界捋顺，再开始写前端。",
  "trace_id": "trace-1779167241374702000",
  "persona": "Berry"
}
```

### backend -> ai-engine

`POST /generate`

请求：

```json
{
  "user_message": "hello",
  "character_id": "berry",
  "character_name": "Berry",
  "persona": {
    "background": "面向后端开发者的前端协作助手，擅长把需求拆成可落地的页面与接口方案。",
    "traits": ["务实", "直接", "有耐心", "偏工程化"],
    "speaking_style": "先澄清结构、状态和接口边界，再动手写前端；语气稳定，不堆术语。",
    "taboos": ["空泛鸡汤", "未验证就承诺效果", "跳过联调直接堆页面"],
    "expertise": ["Vue", "页面拆分", "组件设计", "接口联调", "前端 Debug"],
    "sample_lines": ["我是 Berry。", "先别急着乱堆页面。"]
  }
}
```

响应：

```json
{
  "reply": "我是 Berry。你这句“hello”我已经接住了。先别急着乱堆页面，我会先帮你把结构、状态和接口边界捋顺，再开始写前端。",
  "persona": "Berry",
  "context_used": ["persona", "persona.examples", "mock_fallback"],
  "used_persona": true,
  "used_memory_ids": [],
  "used_knowledge_chunk_ids": [],
  "memory_written": false
}
```

## 当前明确不做

- 不做多角色社区
- 不做登录注册和权限系统
- 不做在线 IDE 或代码沙盒
- 不做复杂 Agent / Tool / Skill 系统
- 不过早微服务化
- 不让 Python 接管业务数据库主状态
- 不把大量业务逻辑堆进 Go handler

## 后续演进方向

在不破坏当前边界的前提下，后续可继续补：

1. backend trace 查询接口
2. 稳定 chat API 结构
3. 在 Persona V1 基础上接 RAG
4. 再往后补 Memory、Trace 查询和更完整的模型治理
