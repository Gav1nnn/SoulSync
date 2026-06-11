<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ChatApiError, fetchRecentMessages, fetchTrace, sendChatMessage } from "../api/chat";
import AgentTaskPanel from "../components/agent/AgentTaskPanel.vue";
import ChatHeader from "../components/chat/ChatHeader.vue";
import MessageComposer from "../components/chat/MessageComposer.vue";
import MessageList from "../components/chat/MessageList.vue";
import StatusPanel from "../components/chat/StatusPanel.vue";
import MemoryPanel from "../components/memory/MemoryPanel.vue";
import WorkspaceConnector from "../components/workspace/WorkspaceConnector.vue";
import type { ChatErrorCode, ChatMessage, ChatStatus, Trace } from "../types/chat";

const quickPrompts = [
  "你好，我叫 Gavin",
  "帮我拆一个用户列表页，包含搜索、表格和空状态",
  "这个项目里 Go 和 Python 分别负责什么",
];

const draft = ref("");
const status = ref<ChatStatus>("idle");
const errorMessage = ref("");
const errorCode = ref<ChatErrorCode | null>(null);
const traceId = ref("");
const contextUsed = ref<string[]>([]);
const usedMemoryIds = ref<string[]>([]);
const memoryWritten = ref(false);
const memoryCandidateCount = ref(0);
const trace = ref<Trace | null>(null);
const messages = ref<ChatMessage[]>([]);

function fillDraft(prompt: string) {
  draft.value = prompt;
}

async function loadTrace(nextTraceId: string) {
  if (!nextTraceId) {
    return;
  }

  const response = await fetchTrace(nextTraceId);
  trace.value = response.trace;
  traceId.value = response.trace.trace_id;
  contextUsed.value = response.trace.context_used;
  usedMemoryIds.value = response.trace.used_memory_ids;
  memoryWritten.value = response.trace.memory_written;
  memoryCandidateCount.value = response.trace.memory_candidate_count;
}

async function restoreRecentMessages() {
  try {
    const response = await fetchRecentMessages(50);
    messages.value = response.messages;

    const latestTraceId = [...response.messages]
      .reverse()
      .find((message) => message.trace_id)?.trace_id;
    if (latestTraceId) {
      traceId.value = latestTraceId;
      try {
        await loadTrace(latestTraceId);
        status.value = "success";
      } catch {
        status.value = "idle";
      }
    }
  } catch (error) {
    status.value = "error";

    if (error instanceof ChatApiError) {
      errorMessage.value = error.message;
      errorCode.value = error.code;
      return;
    }

    errorMessage.value = "最近消息恢复失败，请稍后重试。";
    errorCode.value = "unknown";
  }
}

async function sendMessage() {
  const message = draft.value.trim();
  if (!message || status.value === "sending") {
    return;
  }

  messages.value.push({
    id: `user-${Date.now()}`,
    role: "user",
    content: message,
  });

  draft.value = "";
  status.value = "sending";
  errorMessage.value = "";
  errorCode.value = null;

  try {
    const response = await sendChatMessage(message);

    traceId.value = response.trace_id;
    contextUsed.value = response.context_used;
    usedMemoryIds.value = response.used_memory_ids;
    memoryWritten.value = response.memory_written;
    memoryCandidateCount.value = response.memory_candidate_count;
    trace.value = null;
    messages.value.push({
      id: response.trace_id,
      trace_id: response.trace_id,
      role: "assistant",
      content: response.reply,
    });
    await loadTrace(response.trace_id);
    status.value = "success";
  } catch (error) {
    status.value = "error";

    if (error instanceof ChatApiError) {
      errorMessage.value = error.message;
      errorCode.value = error.code;
      return;
    }

    errorMessage.value = "请求失败，请稍后重试。";
    errorCode.value = "unknown";
  }
}

onMounted(() => {
  void restoreRecentMessages();
});
</script>

<template>
  <main class="chat-page">
    <section class="chat-shell">
      <ChatHeader />

      <div class="workspace">
        <section class="main-column">
          <WorkspaceConnector />
          <AgentTaskPanel />
          <MemoryPanel />

          <section class="quick-prompts" aria-label="quick prompts">
            <button
              v-for="prompt in quickPrompts"
              :key="prompt"
              class="prompt-chip"
              type="button"
              @click="fillDraft(prompt)"
            >
              {{ prompt }}
            </button>
          </section>

          <MessageList :messages="messages" :is-sending="status === 'sending'" />
          <MessageComposer
            v-model="draft"
            :is-sending="status === 'sending'"
            @submit="sendMessage"
          />
        </section>

        <StatusPanel
          :status="status"
          :trace-id="traceId"
          :context-used="contextUsed"
          :used-memory-ids="usedMemoryIds"
          :memory-written="memoryWritten"
          :memory-candidate-count="memoryCandidateCount"
          :trace="trace"
          :error-message="errorMessage"
          :error-code="errorCode"
        />
      </div>
    </section>
  </main>
</template>

<style scoped>
:global(:root) {
  --ink: #17252d;
  --ink-soft: #2b414b;
  --muted: #657780;
  --line: rgba(31, 55, 66, 0.14);
  --line-strong: rgba(31, 55, 66, 0.22);
  --paper: #f5f1e8;
  --panel: rgba(255, 252, 244, 0.78);
  --panel-strong: rgba(255, 252, 244, 0.94);
  --accent: #c85f49;
  --accent-ink: #8d3b31;
  --cyan: #2f7182;
  --ok: #377e5b;
  --warn: #a75f31;
  --danger: #ad332f;
  --radius: 8px;
  --shadow-soft: 0 18px 48px rgba(34, 52, 60, 0.08);
  --shadow-tight: 0 8px 22px rgba(34, 52, 60, 0.08);
  --font-display: "Avenir Next Condensed", "DIN Condensed", "PingFang SC", sans-serif;
  --font-body: "Avenir Next", "PingFang SC", "Hiragino Sans GB", sans-serif;
  --font-mono: "SFMono-Regular", "Menlo", "Monaco", "Courier New", monospace;
}

:global(body) {
  margin: 0;
  min-width: 320px;
  background:
    linear-gradient(90deg, rgba(23, 37, 45, 0.055) 1px, transparent 1px),
    linear-gradient(rgba(23, 37, 45, 0.045) 1px, transparent 1px),
    linear-gradient(135deg, #f4efe4 0%, #eef3ef 56%, #f8f3ea 100%);
  background-size: 28px 28px, 28px 28px, auto;
  color: var(--ink);
  font-family: var(--font-body);
  text-rendering: optimizeLegibility;
}

:global(*) {
  box-sizing: border-box;
}

:global(button),
:global(textarea) {
  font: inherit;
}

.chat-page {
  position: relative;
  min-height: 100vh;
  overflow-x: hidden;
  padding: 16px;
}

.chat-shell {
  position: relative;
  z-index: 1;
  width: min(1440px, 100%);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: shell-in 360ms ease both;
}

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 360px);
  gap: 12px;
  align-items: start;
}

.main-column {
  display: grid;
  gap: 12px;
}

.quick-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background: var(--panel);
  box-shadow: var(--shadow-tight);
}

.prompt-chip {
  min-height: 34px;
  border: 1px solid var(--line);
  border-radius: 999px;
  padding: 0 11px;
  color: var(--ink-soft);
  background: rgba(255, 252, 244, 0.72);
  font-size: 0.84rem;
  font-weight: 800;
  cursor: pointer;
  transition:
    transform 160ms ease,
    border-color 160ms ease,
    background 160ms ease;
}

.prompt-chip:hover {
  transform: translateY(-1px);
  border-color: rgba(200, 95, 73, 0.42);
  background: var(--panel-strong);
}

@media (max-width: 960px) {
  .workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .chat-page {
    padding: 10px;
  }

  .quick-prompts {
    overflow-x: auto;
    flex-wrap: nowrap;
    padding-bottom: 4px;
  }

  .prompt-chip {
    flex: 0 0 auto;
    max-width: 86vw;
    white-space: nowrap;
  }
}

@keyframes shell-in {
  from {
    opacity: 0;
    transform: translateY(12px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
