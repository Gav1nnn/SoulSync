<script setup lang="ts">
import type { ChatMessage } from "../../types/chat";

defineProps<{
  messages: ChatMessage[];
  isSending: boolean;
}>();
</script>

<template>
  <section class="message-list" aria-label="chat messages">
    <article v-if="messages.length === 0" class="empty-state">
      <p class="empty-title">准备就绪</p>
      <p class="empty-copy">
        现在还没有消息。你可以先丢一个具体需求过来，例如页面结构、组件拆分或接口联调问题。
      </p>
    </article>

    <article
      v-for="message in messages"
      :key="message.id"
      class="message"
      :class="`message-${message.role}`"
    >
      <p class="meta">{{ message.role === "assistant" ? "Berry" : "You" }}</p>
      <p class="content">{{ message.content }}</p>
    </article>

    <article v-if="isSending" class="message message-assistant">
      <p class="meta">Berry</p>
      <p class="content">在想。先别催，主链路已经跑起来了。</p>
    </article>
  </section>
</template>

<style scoped>
.message-list {
  display: grid;
  gap: 14px;
  align-content: start;
}

.empty-state,
.message {
  padding: 18px;
  border-radius: 20px;
  border: 1px solid rgba(126, 67, 81, 0.12);
  background: #fffdfb;
}

.empty-state {
  display: grid;
  gap: 8px;
}

.empty-title,
.empty-copy,
.meta,
.content {
  margin: 0;
}

.empty-title,
.meta {
  color: #865865;
  font-size: 0.8rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.empty-copy,
.content {
  line-height: 1.6;
  white-space: pre-wrap;
}

.message {
  max-width: 82%;
  box-shadow: 0 10px 24px rgba(82, 45, 54, 0.06);
}

.message-assistant {
  justify-self: start;
}

.message-user {
  justify-self: end;
  background: #f2dbe2;
  border-color: rgba(126, 67, 81, 0.2);
}

.content {
  margin-top: 8px;
}

@media (max-width: 720px) {
  .message {
    max-width: 100%;
  }
}
</style>
