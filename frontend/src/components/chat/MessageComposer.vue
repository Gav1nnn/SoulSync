<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  modelValue: string;
  isSending: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
  submit: [];
}>();

const canSend = computed(
  () => props.modelValue.trim().length > 0 && !props.isSending,
);

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== "Enter" || event.shiftKey) {
    return;
  }

  event.preventDefault();
  if (canSend.value) {
    emit("submit");
  }
}
</script>

<template>
  <form class="composer" @submit.prevent="emit('submit')">
    <div class="composer-head">
      <label class="label" for="message">给 Berry 的任务卡</label>
    </div>
    <textarea
      id="message"
      class="input"
      rows="4"
      :value="modelValue"
      placeholder="比如：帮我先搭一个用户列表页，包含搜索、表格、空状态和接口联调注意点。"
      @input="
        emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)
      "
      @keydown="handleKeydown"
    />

    <div class="actions">
      <p class="hint">接口、字段、页面目标或报错都可以直接写在这里。</p>
      <button class="send" type="submit" :disabled="!canSend">
        <span>{{ isSending ? "联调中..." : "发送给 Berry" }}</span>
      </button>
    </div>
  </form>
</template>

<style scoped>
.composer {
  position: sticky;
  bottom: 12px;
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background: var(--panel-strong);
  box-shadow: var(--shadow-soft);
  backdrop-filter: blur(12px);
}

.composer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.label,
.hint {
  margin: 0;
}

.label {
  color: var(--ink);
  font-size: 0.92rem;
  font-weight: 900;
}

.input {
  width: 100%;
  min-height: 108px;
  padding: 13px 14px;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background:
    linear-gradient(rgba(255, 255, 255, 0.72), rgba(255, 255, 255, 0.72)),
    repeating-linear-gradient(to bottom, transparent 0, transparent 31px, rgba(31, 55, 66, 0.05) 32px);
  color: var(--ink);
  line-height: 1.6;
  resize: vertical;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background 160ms ease;
}

.input:focus {
  outline: 0;
  border-color: rgba(200, 95, 73, 0.62);
  background: var(--panel-strong);
  box-shadow: 0 0 0 3px rgba(200, 95, 73, 0.12);
}

.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hint {
  color: var(--muted);
  font-size: 0.88rem;
  line-height: 1.55;
}

.send {
  min-width: 120px;
  min-height: 40px;
  padding: 0 16px;
  border: 0;
  border-radius: var(--radius);
  color: #fffaf1;
  background: var(--ink);
  box-shadow: var(--shadow-tight);
  font-weight: 900;
  cursor: pointer;
  transition:
    transform 160ms ease,
    opacity 160ms ease,
    box-shadow 160ms ease;
}

.send:hover:not(:disabled) {
  transform: translateY(-1px);
  background: var(--accent);
  box-shadow: var(--shadow-soft);
}

.send:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  box-shadow: none;
}

@media (max-width: 720px) {
  .composer {
    position: static;
  }

  .composer-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .actions {
    flex-direction: column;
    align-items: stretch;
  }

  .send {
    width: 100%;
  }
}
</style>
