<script setup lang="ts">
import { computed } from "vue";
import type { ChatErrorCode, ChatStatus, Trace } from "../../types/chat";

const props = defineProps<{
  status: ChatStatus;
  traceId: string;
  contextUsed: string[];
  usedMemoryIds: string[];
  memoryWritten: boolean;
  memoryCandidateCount: number;
  trace: Trace | null;
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

const effectiveContextUsed = computed(() => props.trace?.context_used ?? props.contextUsed);
const effectiveUsedMemoryIds = computed(() => props.trace?.used_memory_ids ?? props.usedMemoryIds);
const effectiveMemoryWritten = computed(() => props.trace?.memory_written ?? props.memoryWritten);
const effectiveMemoryCandidateCount = computed(
  () => props.trace?.memory_candidate_count ?? props.memoryCandidateCount,
);
const effectiveTraceId = computed(() => props.trace?.trace_id ?? props.traceId);
const usedKnowledgeChunkIds = computed(() => props.trace?.used_knowledge_chunk_ids ?? []);

const contextGroups = computed(() => {
  const known = new Set(effectiveContextUsed.value);
  return [
    { label: "Persona", active: known.has("persona") },
    { label: "RAG", active: known.has("knowledge") },
    { label: "Memory", active: known.has("memory") },
    { label: "Trace", active: Boolean(effectiveTraceId.value) },
  ];
});

const traceDuration = computed(() => {
  if (!props.trace) {
    return "waiting";
  }
  return `${props.trace.duration_ms} ms`;
});

const traceWindow = computed(() => {
  if (!props.trace) {
    return "waiting";
  }

  return `${formatTraceTime(props.trace.started_at)} -> ${formatTraceTime(props.trace.finished_at)}`;
});

function formatTraceTime(value: string) {
  if (!value) {
    return "unknown";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
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
      <p class="value code">{{ effectiveTraceId || "waiting" }}</p>
      <p class="detail">Duration: {{ traceDuration }}</p>
      <p class="detail code">{{ traceWindow }}</p>
    </section>

    <section class="status-card">
      <p class="label">Context Used</p>
      <div class="tag-list" v-if="effectiveContextUsed.length">
        <span v-for="item in effectiveContextUsed" :key="item">{{ item }}</span>
      </div>
      <p v-else class="value code">waiting</p>
      <p class="detail">确认本轮是否走到 persona、knowledge、memory 或 fallback。</p>
    </section>

    <section class="status-card">
      <p class="label">Memory</p>
      <p class="value code">
        {{ effectiveMemoryWritten ? "written" : "not written" }} / candidates
        {{ effectiveMemoryCandidateCount }}
      </p>
      <p class="detail">
        Used:
        {{ effectiveUsedMemoryIds.length ? effectiveUsedMemoryIds.join(", ") : "none" }}
      </p>
    </section>

    <section class="status-card">
      <p class="label">Knowledge</p>
      <div class="tag-list" v-if="usedKnowledgeChunkIds.length">
        <span v-for="item in usedKnowledgeChunkIds" :key="item">{{ item }}</span>
      </div>
      <p v-else class="value code">none</p>
      <p class="detail">本轮 RAG 命中的知识 chunk id，用来回查 docs 上下文。</p>
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
  top: 12px;
  display: grid;
  gap: 10px;
}

.status-card {
  position: relative;
  display: grid;
  gap: 7px;
  padding: 13px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background: var(--panel);
  box-shadow: var(--shadow-tight);
  backdrop-filter: blur(12px);
}

.hero-card {
  min-height: 118px;
  background: var(--panel-strong);
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
  color: var(--accent);
  font-size: 0.74rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.value {
  color: var(--ink);
  font-size: 1.02rem;
  font-weight: 900;
  overflow-wrap: anywhere;
}

.detail {
  color: var(--muted);
  line-height: 1.5;
  font-size: 0.88rem;
}

.code {
  font-family: var(--font-mono);
  font-size: 0.82rem;
}

.error {
  color: var(--danger);
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: #9aa8ac;
  box-shadow: 0 0 0 6px rgba(154, 168, 172, 0.14);
}

.tone-success .status-dot {
  background: var(--ok);
  box-shadow: 0 0 0 6px rgba(93, 156, 123, 0.16);
}

.tone-sending .status-dot {
  background: var(--accent);
  box-shadow: 0 0 0 6px rgba(211, 111, 85, 0.16);
  animation: pulse 900ms ease-in-out infinite;
}

.tone-error .status-dot {
  background: var(--danger);
  box-shadow: 0 0 0 6px rgba(179, 61, 55, 0.16);
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.module-pill,
.tag-list span {
  border: 1px solid var(--line);
  border-radius: 999px;
  padding: 7px 9px;
  color: var(--muted);
  background: rgba(255, 255, 255, 0.48);
  font-size: 0.8rem;
  font-weight: 800;
  text-align: center;
}

.module-pill.active {
  color: var(--ink);
  border-color: rgba(200, 95, 73, 0.32);
  background: #fff0e7;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.tag-list span {
  color: var(--cyan);
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
