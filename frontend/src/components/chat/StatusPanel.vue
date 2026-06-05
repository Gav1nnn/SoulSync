<script setup lang="ts">
import { computed } from "vue";
import type { ChatErrorCode, ChatStatus } from "../../types/chat";

const props = defineProps<{
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

const statusTone = computed(() => `tone-${props.status}`);

const contextGroups = computed(() => {
  const known = new Set(props.contextUsed);
  return [
    { label: "Persona", active: known.has("persona") },
    { label: "RAG", active: known.has("knowledge") },
    { label: "Memory", active: known.has("memory") },
    { label: "Trace", active: Boolean(props.traceId) },
  ];
});
</script>

<template>
  <aside class="status-panel">
    <section class="status-card hero-card" :class="statusTone">
      <div class="status-row">
        <p class="label">Live Chain</p>
        <span class="status-dot" aria-hidden="true"></span>
      </div>
      <p class="value">{{ statusLabels[status] }}</p>
      <p class="detail">
        <template v-if="status === 'idle'">
          输入问题后，会依次经过 frontend、backend、ai-engine。
        </template>
        <template v-else-if="status === 'sending'">
          正在组织 Persona、RAG、Memory 并调用模型。
        </template>
        <template v-else-if="status === 'success'">
          当前请求已返回，可以继续验证记忆、知识库和人设。
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
      <p class="label">Modules</p>
      <div class="module-grid">
        <span
          v-for="item in contextGroups"
          :key="item.label"
          class="module-pill"
          :class="{ active: item.active }"
        >
          {{ item.label }}
        </span>
      </div>
    </section>

    <section class="status-card">
      <p class="label">Trace</p>
      <p class="value code">{{ traceId || "waiting" }}</p>
      <p class="detail">最近一次请求 ID，用来定位后端日志和上下文使用情况。</p>
    </section>

    <section class="status-card">
      <p class="label">Context Used</p>
      <div class="tag-list" v-if="contextUsed.length">
        <span v-for="item in contextUsed" :key="item">{{ item }}</span>
      </div>
      <p v-else class="value code">waiting</p>
      <p class="detail">确认本轮是否走到 persona、knowledge、memory 或 fallback。</p>
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
  position: sticky;
  top: 18px;
  display: grid;
  gap: 12px;
}

.status-card {
  position: relative;
  display: grid;
  gap: 8px;
  padding: 16px;
  overflow: hidden;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 22px;
  background: rgba(255, 253, 248, 0.78);
  box-shadow: 0 14px 34px rgba(43, 76, 88, 0.07);
  backdrop-filter: blur(16px);
}

.hero-card {
  min-height: 142px;
  background:
    radial-gradient(circle at 88% 18%, rgba(255, 211, 191, 0.76), transparent 8rem),
    linear-gradient(145deg, rgba(255, 253, 248, 0.92), rgba(240, 249, 249, 0.82));
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.label,
.value,
.detail {
  margin: 0;
}

.label {
  color: #d36f55;
  font-size: 0.74rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.value {
  color: #20333c;
  font-size: 1.02rem;
  font-weight: 900;
  overflow-wrap: anywhere;
}

.detail {
  color: #60717a;
  line-height: 1.5;
  font-size: 0.88rem;
}

.code {
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  font-size: 0.82rem;
}

.error {
  color: #b33d37;
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: #9aa8ac;
  box-shadow: 0 0 0 6px rgba(154, 168, 172, 0.14);
}

.tone-success .status-dot {
  background: #5d9c7b;
  box-shadow: 0 0 0 6px rgba(93, 156, 123, 0.16);
}

.tone-sending .status-dot {
  background: #d36f55;
  box-shadow: 0 0 0 6px rgba(211, 111, 85, 0.16);
  animation: pulse 900ms ease-in-out infinite;
}

.tone-error .status-dot {
  background: #b33d37;
  box-shadow: 0 0 0 6px rgba(179, 61, 55, 0.16);
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.module-pill,
.tag-list span {
  border: 1px solid rgba(43, 76, 88, 0.1);
  border-radius: 999px;
  padding: 7px 9px;
  color: #728189;
  background: rgba(255, 255, 255, 0.48);
  font-size: 0.8rem;
  font-weight: 800;
  text-align: center;
}

.module-pill.active {
  color: #20333c;
  border-color: rgba(211, 111, 85, 0.32);
  background: #fff0e7;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.tag-list span {
  color: #315f70;
  text-align: left;
}

@media (max-width: 960px) {
  .status-panel {
    position: static;
  }
}

@keyframes pulse {
  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.18);
  }
}
</style>
