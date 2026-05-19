<script setup lang="ts">
import { computed, ref } from "vue";

type Role = "user" | "assistant";

type ChatMessage = {
  id: string;
  role: Role;
  content: string;
};

type ChatResponse = {
  reply: string;
  trace_id: string;
  persona: string;
};

const draft = ref("");
const isSending = ref(false);
const errorMessage = ref("");
const traceId = ref("");
const messages = ref<ChatMessage[]>([
  {
    id: "welcome",
    role: "assistant",
    content:
      "Berry 在。先把需求说清楚，我帮你把前端这块补起来，别再拿后端思维硬怼页面了。",
  },
]);

const canSend = computed(() => draft.value.trim().length > 0 && !isSending.value);

async function sendMessage() {
  const message = draft.value.trim();
  if (!message || isSending.value) {
    return;
  }

  errorMessage.value = "";
  messages.value.push({
    id: `user-${Date.now()}`,
    role: "user",
    content: message,
  });
  draft.value = "";
  isSending.value = true;

  try {
    const response = await fetch("/api/chat", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ message }),
    });

    const payload = (await response.json()) as Partial<ChatResponse> & {
      error?: string;
    };

    if (!response.ok || !payload.reply || !payload.trace_id) {
      throw new Error(payload.error || "Berry 这边没回上来，先检查后端和 AI 服务。");
    }

    traceId.value = payload.trace_id;
    messages.value.push({
      id: payload.trace_id,
      role: "assistant",
      content: payload.reply,
    });
  } catch (error) {
    const messageText =
      error instanceof Error ? error.message : "请求失败，请稍后重试。";
    errorMessage.value = messageText;
  } finally {
    isSending.value = false;
  }
}
</script>

<template>
  <main class="chat-page">
    <section class="chat-shell">
      <header class="hero">
        <p class="eyebrow">SoulSync / Berry</p>
        <h1>最小对话闭环</h1>
        <p class="subtitle">
          前端只负责交互，Go 串主流程，Python 负责 Berry 的回复组织。现在先把这条链路跑通。
        </p>
      </header>

      <section class="messages" aria-label="chat messages">
        <article
          v-for="message in messages"
          :key="message.id"
          class="message"
          :class="`message-${message.role}`"
        >
          <p class="role">{{ message.role === "assistant" ? "Berry" : "You" }}</p>
          <p class="content">{{ message.content }}</p>
        </article>

        <article v-if="isSending" class="message message-assistant">
          <p class="role">Berry</p>
          <p class="content">在想。你先别催，链路已经在跑了。</p>
        </article>
      </section>

      <p v-if="traceId" class="trace">trace: {{ traceId }}</p>
      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <form class="composer" @submit.prevent="sendMessage">
        <label class="sr-only" for="message">message</label>
        <textarea
          id="message"
          v-model="draft"
          class="input"
          rows="4"
          placeholder="比如：帮我先搭一个用户列表页，包含搜索和表格。"
        />
        <button class="send" type="submit" :disabled="!canSend">
          {{ isSending ? "发送中..." : "发送" }}
        </button>
      </form>
    </section>
  </main>
</template>

<style scoped>
:global(body) {
  margin: 0;
  background:
    radial-gradient(circle at top, rgba(255, 208, 214, 0.9), transparent 32%),
    linear-gradient(180deg, #fff8f2 0%, #fde8e4 100%);
  color: #2b1b1f;
  font-family: "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
}

:global(*) {
  box-sizing: border-box;
}

.chat-page {
  min-height: 100vh;
  padding: 32px 16px;
}

.chat-shell {
  width: min(920px, 100%);
  margin: 0 auto;
  display: grid;
  gap: 20px;
}

.hero {
  padding: 28px;
  border: 1px solid rgba(126, 67, 81, 0.14);
  border-radius: 28px;
  background: rgba(255, 252, 250, 0.82);
  backdrop-filter: blur(12px);
  box-shadow: 0 18px 45px rgba(126, 67, 81, 0.1);
}

.eyebrow {
  margin: 0 0 10px;
  color: #a55368;
  font-size: 0.82rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3.2rem);
  line-height: 1;
}

.subtitle {
  margin: 12px 0 0;
  max-width: 54rem;
  color: #5a4047;
  line-height: 1.6;
}

.messages {
  display: grid;
  gap: 14px;
}

.message {
  max-width: 80%;
  padding: 16px 18px;
  border-radius: 22px;
  box-shadow: 0 12px 28px rgba(79, 43, 53, 0.08);
}

.message-assistant {
  justify-self: start;
  background: #fffaf8;
  border: 1px solid rgba(165, 83, 104, 0.14);
}

.message-user {
  justify-self: end;
  background: #a55368;
  color: #fff9fb;
}

.role,
.content,
.trace,
.error {
  margin: 0;
}

.role {
  font-size: 0.78rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  opacity: 0.72;
}

.content {
  margin-top: 8px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.trace {
  color: #81545f;
  font-size: 0.88rem;
}

.error {
  color: #a12239;
  font-size: 0.92rem;
}

.composer {
  display: grid;
  gap: 12px;
  padding: 20px;
  border: 1px solid rgba(126, 67, 81, 0.12);
  border-radius: 24px;
  background: rgba(255, 252, 250, 0.88);
  box-shadow: 0 18px 45px rgba(126, 67, 81, 0.08);
}

.input {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid rgba(126, 67, 81, 0.18);
  border-radius: 18px;
  background: #fffdfc;
  color: inherit;
  font: inherit;
  resize: vertical;
}

.input:focus {
  outline: 2px solid rgba(165, 83, 104, 0.25);
  border-color: #a55368;
}

.send {
  justify-self: end;
  min-width: 132px;
  padding: 12px 18px;
  border: 0;
  border-radius: 999px;
  background: linear-gradient(135deg, #a55368 0%, #db7d77 100%);
  color: #fffdfd;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.send:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 720px) {
  .chat-page {
    padding: 20px 12px;
  }

  .hero,
  .composer {
    padding: 18px;
    border-radius: 22px;
  }

  .message {
    max-width: 100%;
  }

  .send {
    width: 100%;
    justify-self: stretch;
  }
}
</style>
