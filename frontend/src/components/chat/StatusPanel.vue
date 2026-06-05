<script setup lang="ts">
import type { ChatErrorCode, ChatStatus } from "../../types/chat";

defineProps<{
  status: ChatStatus;
  traceId: string;
  contextUsed: string[];
  usedMemoryIds: string[];
  memoryWritten: boolean;
  memoryCandidateCount: number;
  errorMessage: string;
  errorCode: ChatErrorCode | null;
}>();

const statusLabels: Record<ChatStatus, string> = {
  idle: "等待输入",
  sending: "请求发送中",
  success: "链路正常",
  error: "请求失败",
};
</script>

<template>
  <aside class="status-panel">
    <section class="status-card">
      <p class="label">状态</p>
      <p class="value">{{ statusLabels[status] }}</p>
      <p class="detail">
        <template v-if="status === 'idle'">
          前端外壳已就绪，等待发起第一条请求。
        </template>
        <template v-else-if="status === 'sending'">
          frontend -> backend -> ai-engine 正在执行。
        </template>
        <template v-else-if="status === 'success'">
          当前这轮请求已经顺利返回，可以继续围绕这套外壳扩功能。
        </template>
        <template v-else-if="errorCode === 'service_unavailable' || errorCode === 'network'">
          backend 或 ai-engine 当前不可用，优先检查服务进程和端口。
        </template>
        <template v-else>
          请求已经失败，请根据错误信息定位输入、接口或服务问题。
        </template>
      </p>
    </section>

    <section class="status-card">
      <p class="label">Trace</p>
      <p class="value code">{{ traceId || "waiting" }}</p>
      <p class="detail">先展示最近一次 trace_id，后续可以在这里接完整 trace 详情。</p>
    </section>

    <section class="status-card">
      <p class="label">Context Used</p>
      <p class="value code">{{ contextUsed.length ? contextUsed.join(", ") : "waiting" }}</p>
      <p class="detail">这里展示本轮实际注入的上下文，方便确认有没有走到 persona、knowledge 或 fallback。</p>
    </section>

    <section class="status-card">
      <p class="label">Memory</p>
      <p class="value code">{{ memoryWritten ? "written" : "not written" }} / candidates {{ memoryCandidateCount }}</p>
      <p class="detail">
        Used:
        {{ usedMemoryIds.length ? usedMemoryIds.join(", ") : "none" }}
      </p>
    </section>

    <section class="status-card">
      <p class="label">Error</p>
      <p class="value error">{{ errorMessage || "none" }}</p>
      <p class="detail">这里集中展示当前错误，避免把失败状态散落在页面不同位置。</p>
    </section>
  </aside>
</template>

<style scoped>
.status-panel {
  display: grid;
  gap: 12px;
}

.status-card {
  display: grid;
  gap: 6px;
  padding: 16px;
  border: 1px solid rgba(126, 67, 81, 0.12);
  border-radius: 18px;
  background: #fffaf8;
}

.label,
.value,
.detail {
  margin: 0;
}

.label {
  color: #875b67;
  font-size: 0.78rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.value {
  font-size: 1rem;
  font-weight: 700;
}

.detail {
  color: #654e54;
  line-height: 1.5;
  font-size: 0.9rem;
}

.code {
  font-family: "SFMono-Regular", Consolas, monospace;
  font-size: 0.92rem;
}

.error {
  color: #9f2138;
}
</style>
