# Current Status

## 当前目标

SoulSync 当前阶段只做一件事：打通最小主链路，形成“可运行、可演示”的闭环。

```text
用户提问
-> frontend 收集输入
-> backend 接收请求并串联主流程
-> ai-engine 返回 Berry 风格回复
-> backend 记录最小 trace
-> frontend 展示结果
```

## 已完成

- 初始化 `frontend`
- 初始化 `backend`
- 初始化 `ai-engine`
- 三端最小 `health check`
- 前端最小聊天页
- 前端通过 `/api/chat` 请求 Go
- Go 通过 HTTP 调用 Python `POST /generate`
- Python 已升级为 Persona V1：
  - 结构化 Berry persona
  - few-shot 风格示例
  - Ollama / DeepSeek 双 provider
  - mock fallback
- Python 已接入 RAG V1：
  - 只检索 `docs/`
  - Ollama `qwen3-embedding:0.6b` + 关键词混合检索
  - embedding 结果会缓存到本地索引文件
  - 命中时返回 `used_knowledge_chunk_ids`
  - 检索失败时不阻断主链路
- Go 生成 `trace_id`
- Go 记录最小内存态 trace
- 前后端联调已通过

## 当前可演示内容

- 浏览器打开前端页面后，可以输入一句话
- 前端会请求 Go 服务
- Go 服务会请求 AI 服务
- AI 服务返回 Berry 风格回复
- 前端展示回复内容和最近一次 `trace_id`

当前这一步已经满足：

- 可运行
- 可验证
- 可演示

## 当前未做

- 数据库
- 消息持久化
- 长期 Memory
- Trace 查询接口
- 在线 IDE / 沙盒

## 限制

- trace 目前只保存在内存里，服务重启后会丢失
- 未配置模型时，Berry 回复会回退到 persona mock
- 当前 RAG 只覆盖仓库 `docs/`，未接 rerank 和外部向量数据库
- 当前接口结构还处于早期阶段，后续还会继续收敛

## 下一步计划

建议按下面顺序推进：

1. 给 backend 增加结构化 Memory 主存储
2. 补最小 Trace 查询接口
3. 再补 Memory 提取和注入链路
4. 再往后补 rerank / 外部向量库 / 更完整的评测

## 维护规则

这里主要回答两个问题：

- 当前做到哪了
- 下一步最应该做什么
