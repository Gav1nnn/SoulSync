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
      <span class="empty-mark" aria-hidden="true">Berry</span>
      <p class="empty-title">把需求丢过来</p>
      <p class="empty-copy">
        先从一个小问题开始：页面结构、组件拆分、接口联调、Bug 排查都可以。记忆和知识库会在右侧同步亮起来。
      </p>
      <div class="empty-lines" aria-hidden="true">
        <span></span>
        <span></span>
        <span></span>
      </div>
    </article>

    <article
      v-for="message in messages"
      :key="message.id"
      class="message"
      :class="`message-${message.role}`"
    >
      <p class="meta">
        <span>{{ message.role === "assistant" ? "Berry" : "You" }}</span>
        <span>{{ message.role === "assistant" ? "Frontend Senpai" : "Backend Dev" }}</span>
      </p>
      <p class="content">{{ message.content }}</p>
    </article>

    <article v-if="isSending" class="message message-assistant">
      <p class="meta">
        <span>Berry</span>
        <span>Thinking</span>
      </p>
      <p class="content loading-copy">
        <span>在翻记忆和资料</span>
        <span class="typing-dots" aria-hidden="true"><i></i><i></i><i></i></span>
      </p>
    </article>
  </section>
</template>

<style scoped>
.message-list {
  display: grid;
  min-height: 470px;
  max-height: 62vh;
  gap: 16px;
  align-content: start;
  overflow: auto;
  padding: 18px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 30px;
  background:
    linear-gradient(rgba(255, 253, 248, 0.84), rgba(255, 253, 248, 0.7)),
    repeating-linear-gradient(
      to bottom,
      transparent 0,
      transparent 31px,
      rgba(43, 76, 88, 0.055) 32px
    );
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82);
  scrollbar-width: thin;
}

.empty-state,
.message {
  position: relative;
  padding: 18px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  background: rgba(255, 253, 248, 0.92);
}

.empty-state {
  display: grid;
  gap: 10px;
  min-height: 260px;
  place-content: center;
  overflow: hidden;
  border-radius: 28px;
  text-align: center;
}

.empty-state::before {
  position: absolute;
  inset: 18px;
  content: "";
  border: 1px dashed rgba(221, 120, 91, 0.24);
  border-radius: 22px;
}

.empty-mark {
  justify-self: center;
  display: grid;
  place-items: center;
  width: 76px;
  height: 76px;
  border-radius: 28px;
  color: #20333c;
  background: linear-gradient(145deg, #ffd3bf, #b9d8df);
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.empty-title,
.empty-copy,
.meta,
.content {
  margin: 0;
}

.empty-title,
.meta {
  color: #d36f55;
  font-size: 0.76rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.empty-title {
  color: #20333c;
  font-size: 1.3rem;
  letter-spacing: -0.02em;
  text-transform: none;
}

.empty-copy,
.content {
  line-height: 1.6;
  white-space: pre-wrap;
}

.empty-copy {
  max-width: 34rem;
  color: #60717a;
}

.empty-lines {
  display: grid;
  gap: 7px;
  justify-self: center;
  width: min(280px, 72vw);
  margin-top: 8px;
}

.empty-lines span {
  height: 7px;
  border-radius: 999px;
  background: rgba(43, 76, 88, 0.08);
}

.empty-lines span:nth-child(2) {
  width: 82%;
}

.empty-lines span:nth-child(3) {
  width: 58%;
}

.message {
  max-width: 82%;
  border-radius: 24px;
  box-shadow: 0 16px 34px rgba(43, 76, 88, 0.08);
  animation: message-in 260ms ease both;
}

.message-assistant {
  justify-self: start;
  border-bottom-left-radius: 8px;
}

.message-user {
  justify-self: end;
  color: #20333c;
  border-color: rgba(221, 120, 91, 0.22);
  border-bottom-right-radius: 8px;
  background: linear-gradient(145deg, #ffe7d8, #fff8f1);
}

.content {
  margin-top: 8px;
  color: #283e48;
}

.meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.meta span:last-child {
  color: rgba(43, 76, 88, 0.42);
}

.loading-copy {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.typing-dots {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.typing-dots i {
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: #d36f55;
  animation: dot-bounce 900ms ease-in-out infinite;
}

.typing-dots i:nth-child(2) {
  animation-delay: 120ms;
}

.typing-dots i:nth-child(3) {
  animation-delay: 240ms;
}

@media (max-width: 720px) {
  .message-list {
    min-height: 420px;
    max-height: none;
    padding: 12px;
  }

  .message {
    max-width: 100%;
  }
}

@keyframes message-in {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.99);
  }

  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes dot-bounce {
  0%,
  80%,
  100% {
    transform: translateY(0);
    opacity: 0.35;
  }

  40% {
    transform: translateY(-4px);
    opacity: 1;
  }
}
</style>
