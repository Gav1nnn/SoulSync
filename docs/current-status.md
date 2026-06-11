# Current Status

## 当前目标

SoulSync 正在从聊天助手推进到日常可用的前端协作 Agent。

目标闭环：

```text
连接本地项目
-> 扫描项目结构、接口、前端入口和项目文档
-> Berry 结合 persona、memory、RAG 和 workspace summary 生成短计划
-> Go 执行受限 ReAct action
-> 独立任务分支写入前端代码
-> 运行白名单验证命令
-> 前端展示计划、步骤、改动文件、验证结果和失败原因
```

## 已完成

- 聊天主链路
  - Vue 聊天页请求 Go `/api/chat`
  - Go 调 Python `/generate`
  - Python 返回 Berry 风格回复
  - Go 保存消息、Memory、Trace
  - 前端刷新后恢复最近消息

- Persona / Memory / RAG / Trace
  - Berry persona 固定为前端协作学姐，不做多角色
  - Memory V1 支持保存、注入、禁用和重新启用
  - RAG 检索 SoulSync `docs/`
  - Workspace 扫描会发现目标项目 README/docs/API 文档，并把文档片段注入 Agent 上下文
  - Trace 查询接口可查看完整聊天 trace

- Workspace
  - `POST /api/workspaces` 连接本地绝对路径
  - 校验路径存在、是 git 仓库、是绝对路径
  - `GET /api/workspaces/current` 展示路径、分支和 dirty 状态
  - `GET /api/workspaces/current/summary` 扫描目录树、包管理器、前后端框架、Gin 接口候选、前端入口、API client、类型文件、项目文档和验证命令

- Agent Task
  - `POST /api/agent/tasks` 创建任务
  - `GET /api/agent/tasks/:id` 查看任务详情
  - `GET /api/agent/tasks` 查看最近任务，前端刷新后可恢复最近任务
  - `POST /api/agent/tasks/:id/retry` 支持失败任务基于同一分支重试
  - 状态机覆盖 `queued -> planning -> running -> verifying -> completed|failed`
  - 每个任务创建独立分支 `agent/frontend-from-api-YYYYMMDD-HHMMSS`
  - Go 限制文件读写在 workspace 根目录内
  - Go 只执行扫描识别出的白名单验证命令
  - Python 负责 planner 和 stepper，Go 负责 action 执行

- 接口到页面闭环
  - Gin route/handler/request/response struct 候选识别
  - Agent 可生成前端类型、API client 和 Vue 页面
  - 任务结果展示分支名、改动文件、验证输出、失败文件和下一步建议

- 评测
  - `ai-engine/evals/` 包含 fixture 项目和固定 Agent 任务
  - 当前固定任务集覆盖 5 个接口到页面 case
  - 本地 eval 覆盖 Task Completion、Tool Correctness、Plan Adherence、Plan Quality、Frontend Quality、RAG Relevancy、Faithfulness、Persona/Memory Safety
  - 默认设置 DeepEval telemetry opt-out

## 当前可演示内容

1. 启动 ai-engine、backend、frontend。
2. 在前端连接一个本地 git 项目。
3. 查看 workspace 扫描结果和项目文档片段。
4. 创建“根据用户列表接口生成前端页面”任务。
5. 观察 Agent 计划、读文件、写文件、验证和结果。
6. 失败时可在同一任务分支重试。
7. 刷新页面后可从最近任务列表恢复任务详情。

## 当前边界

- 不做登录注册和权限系统。
- 不做多角色社区。
- 不做在线 IDE 或代码沙盒。
- 不引入复杂 Agent/Tool/Skill 平台。
- 不接外部向量数据库，项目文档检索先保持本地扫描和片段注入。
- 不让 Python 接管业务主状态；任务、workspace、memory、trace 的主流程仍由 Go 管理。

## 剩余工作

- 强化真实项目上的接口识别覆盖率，尤其是更复杂的 Gin 分组、middleware 和多文件 handler。
- 改善生成代码的项目风格贴合度，例如路由注册、组件复用和现有 API client 约定。
- 给 Agent task 增加更完整的 trace 视图，把 action、observation、context_used 和耗时统一到 Trace 面板。
- 继续扩充 fixture 和 eval case 的复杂度，覆盖失败恢复、分页/筛选、嵌套响应、表单提交等常见页面。
- 继续打磨前端工作台，把 Workspace、Agent、Trace、Memory 的状态切换做得更稳定。

## 验收命令

```bash
cd backend && go test ./...
cd ai-engine && uv run python -m unittest discover -s tests -p 'test_*.py'
cd ai-engine && uv run python -m evals.run_agent_eval
cd frontend && npm run build
```
