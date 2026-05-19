# SoulSync Docs

## 文档列表

- [current-status.md](./current-status.md)
  - 记录当前阶段做到了什么、没做到什么、下一步建议是什么
- [architecture.md](./architecture.md)
  - 记录项目定位、三端职责、调用链路、接口边界
- [runbook.md](./runbook.md)
  - 记录本地启动、测试、联调、常用命令

## 维护建议

- 开发进度变化时，优先更新 `current-status.md`
- 接口或职责边界变化时，优先更新 `architecture.md`
- 启动方式、测试命令、联调步骤变化时，优先更新 `runbook.md`

## 当前阶段

当前项目已经完成最小可运行闭环：

```text
frontend -> backend -> ai-engine -> backend -> frontend
```

后续开发仍然暂时围绕主链路推进，不主动扩展无关系统。
