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
</script>

<template>
  <form class="composer" @submit.prevent="emit('submit')">
    <label class="label" for="message">输入你的问题</label>
    <textarea
      id="message"
      class="input"
      rows="4"
      :value="modelValue"
      placeholder="比如：帮我先搭一个用户列表页，包含搜索、表格和空状态。"
      @input="
        emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)
      "
    />

    <div class="actions">
      <p class="hint">当前只做最小闭环，不展开复杂业务逻辑。</p>
      <button class="send" type="submit" :disabled="!canSend">
        {{ isSending ? "发送中..." : "发送" }}
      </button>
    </div>
  </form>
</template>

<style scoped>
.composer {
  display: grid;
  gap: 12px;
  padding: 20px;
  border: 1px solid rgba(126, 67, 81, 0.12);
  border-radius: 24px;
  background: #fffaf8;
}

.label,
.hint {
  margin: 0;
}

.label {
  font-size: 0.92rem;
  font-weight: 600;
}

.input {
  width: 100%;
  min-height: 120px;
  padding: 14px 16px;
  border: 1px solid rgba(126, 67, 81, 0.18);
  border-radius: 18px;
  background: #fffdfc;
  color: inherit;
  font: inherit;
  resize: vertical;
}

.input:focus {
  outline: 2px solid rgba(165, 83, 104, 0.2);
  border-color: #a55368;
}

.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hint {
  color: #6e545b;
  font-size: 0.88rem;
}

.send {
  min-width: 120px;
  padding: 12px 18px;
  border: 0;
  border-radius: 999px;
  background: #8f5864;
  color: #fffdfd;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.send:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

@media (max-width: 720px) {
  .actions {
    flex-direction: column;
    align-items: stretch;
  }

  .send {
    width: 100%;
  }
}
</style>
