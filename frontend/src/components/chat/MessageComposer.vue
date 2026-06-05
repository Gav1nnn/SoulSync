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
      <span class="shortcut">Enter 发送 / Shift + Enter 换行</span>
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
      <p class="hint">建议描述接口字段、页面目标或当前报错，Berry 会同步使用记忆和资料。</p>
      <button class="send" type="submit" :disabled="!canSend">
        <span>{{ isSending ? "联调中..." : "发送给 Berry" }}</span>
      </button>
    </div>
  </form>
</template>

<style scoped>
.composer {
  position: sticky;
  bottom: 16px;
  display: grid;
  gap: 12px;
  padding: 18px;
  border: 1px solid rgba(43, 76, 88, 0.13);
  border-radius: 28px;
  background: rgba(255, 253, 248, 0.88);
  box-shadow: 0 18px 50px rgba(43, 76, 88, 0.1);
  backdrop-filter: blur(18px);
}

.composer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.label,
.hint,
.shortcut {
  margin: 0;
}

.label {
  color: #20333c;
  font-size: 0.92rem;
  font-weight: 900;
}

.shortcut {
  color: #7b8b92;
  font-size: 0.78rem;
}

.input {
  width: 100%;
  min-height: 120px;
  padding: 15px 16px;
  border: 1px solid rgba(43, 76, 88, 0.16);
  border-radius: 20px;
  background:
    linear-gradient(rgba(255, 255, 255, 0.72), rgba(255, 255, 255, 0.72)),
    repeating-linear-gradient(to bottom, transparent 0, transparent 31px, rgba(43, 76, 88, 0.06) 32px);
  color: inherit;
  line-height: 1.7;
  resize: vertical;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background 160ms ease;
}

.input:focus {
  outline: 0;
  border-color: rgba(211, 111, 85, 0.72);
  background: #fffefb;
  box-shadow: 0 0 0 4px rgba(211, 111, 85, 0.12);
}

.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hint {
  color: #61737b;
  font-size: 0.88rem;
  line-height: 1.55;
}

.send {
  min-width: 120px;
  padding: 13px 18px;
  border: 0;
  border-radius: 999px;
  color: #fffdf8;
  background:
    linear-gradient(135deg, #20333c, #315f70 58%, #d36f55);
  box-shadow: 0 12px 30px rgba(32, 51, 60, 0.18);
  font-weight: 900;
  cursor: pointer;
  transition:
    transform 160ms ease,
    opacity 160ms ease,
    box-shadow 160ms ease;
}

.send:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 16px 34px rgba(32, 51, 60, 0.22);
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
