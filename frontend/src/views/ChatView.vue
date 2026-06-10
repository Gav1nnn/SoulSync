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
    <div class="ambient ambient-one" aria-hidden="true"></div>
    <div class="ambient ambient-two" aria-hidden="true"></div>

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
:global(body) {
  margin: 0;
  min-width: 320px;
  background:
    radial-gradient(circle at 10% 8%, rgba(255, 197, 176, 0.58), transparent 28rem),
    radial-gradient(circle at 86% 14%, rgba(137, 186, 220, 0.36), transparent 24rem),
    linear-gradient(135deg, #fff8ef 0%, #f5efe4 48%, #edf4f1 100%);
  color: #1f2930;
  font-family:
    "Tsukimi Rounded",
    "Yuanti SC",
    "Hiragino Maru Gothic ProN",
    "PingFang SC",
    sans-serif;
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
  overflow: hidden;
  padding: 28px 18px;
}

.chat-page::before {
  position: fixed;
  inset: 0;
  pointer-events: none;
  content: "";
  opacity: 0.28;
  background-image:
    linear-gradient(rgba(35, 53, 62, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(35, 53, 62, 0.05) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: linear-gradient(to bottom, black 0%, transparent 80%);
}

.ambient {
  position: fixed;
  pointer-events: none;
  border-radius: 999px;
  filter: blur(2px);
}

.ambient-one {
  width: 180px;
  height: 180px;
  right: -46px;
  top: 100px;
  border: 1px solid rgba(40, 93, 122, 0.18);
  background: rgba(255, 255, 255, 0.24);
}

.ambient-two {
  width: 140px;
  height: 140px;
  left: -42px;
  bottom: 72px;
  border: 1px dashed rgba(221, 120, 91, 0.28);
}

.chat-shell {
  position: relative;
  z-index: 1;
  width: min(1120px, 100%);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: shell-in 560ms ease both;
}

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(286px, 330px);
  gap: 20px;
  align-items: start;
}

.main-column {
  display: grid;
  gap: 16px;
}

.quick-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.prompt-chip {
  border: 1px solid rgba(43, 76, 88, 0.13);
  border-radius: 999px;
  padding: 9px 13px;
  color: #294654;
  background: rgba(255, 253, 248, 0.72);
  box-shadow: 0 10px 26px rgba(42, 61, 69, 0.06);
  cursor: pointer;
  transition:
    transform 160ms ease,
    border-color 160ms ease,
    background 160ms ease;
}

.prompt-chip:hover {
  transform: translateY(-2px);
  border-color: rgba(221, 120, 91, 0.42);
  background: #fffdf8;
}

@media (max-width: 960px) {
  .workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .chat-page {
    padding: 16px 12px;
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
