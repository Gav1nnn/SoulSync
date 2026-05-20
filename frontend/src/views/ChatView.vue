<script setup lang="ts">
import { ref } from "vue";
import { ChatApiError, sendChatMessage } from "../api/chat";
import ChatHeader from "../components/chat/ChatHeader.vue";
import MessageComposer from "../components/chat/MessageComposer.vue";
import MessageList from "../components/chat/MessageList.vue";
import StatusPanel from "../components/chat/StatusPanel.vue";
import type { ChatErrorCode, ChatMessage, ChatStatus } from "../types/chat";

const draft = ref("");
const status = ref<ChatStatus>("idle");
const errorMessage = ref("");
const errorCode = ref<ChatErrorCode | null>(null);
const traceId = ref("");
const messages = ref<ChatMessage[]>([]);

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
    messages.value.push({
      id: response.trace_id,
      role: "assistant",
      content: response.reply,
    });
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
</script>

<template>
  <main class="chat-page">
    <section class="chat-shell">
      <ChatHeader />

      <div class="workspace">
        <section class="main-column">
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
  background: #f6efeb;
  color: #2b1b1f;
  font-family: "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
}

:global(*) {
  box-sizing: border-box;
}

.chat-page {
  min-height: 100vh;
  padding: 24px 16px;
}

.chat-shell {
  width: min(1120px, 100%);
  margin: 0 auto;
  display: grid;
  gap: 18px;
}

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 18px;
  align-items: start;
}

.main-column {
  display: grid;
  gap: 18px;
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
}
</style>
