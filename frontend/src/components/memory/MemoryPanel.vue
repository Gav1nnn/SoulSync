<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { fetchMemories, MemoryApiError, updateMemoryStatus } from "../../api/memory";
import type { Memory, MemoryStatus } from "../../types/memory";

const memories = ref<Memory[]>([]);
const isLoading = ref(false);
const updatingId = ref("");
const errorMessage = ref("");

const activeCount = computed(() => memories.value.filter((memory) => memory.status === "active").length);
const disabledCount = computed(() => memories.value.filter((memory) => memory.status === "disabled").length);

async function loadMemories() {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    const response = await fetchMemories();
    memories.value = response.memories;
  } catch (error) {
    errorMessage.value =
      error instanceof MemoryApiError ? error.message : "Memory 读取失败。";
  } finally {
    isLoading.value = false;
  }
}

async function setMemoryStatus(memory: Memory, status: MemoryStatus) {
  if (updatingId.value) {
    return;
  }

  updatingId.value = memory.id;
  errorMessage.value = "";

  try {
    const response = await updateMemoryStatus(memory.id, status);
    memories.value = memories.value.map((item) =>
      item.id === response.memory.id ? response.memory : item,
    );
  } catch (error) {
    errorMessage.value =
      error instanceof MemoryApiError ? error.message : "Memory 状态更新失败。";
  } finally {
    updatingId.value = "";
  }
}

function formatConfidence(value: number) {
  return `${Math.round(value * 100)}%`;
}

onMounted(() => {
  void loadMemories();
});
</script>

<template>
  <section class="memory-card" aria-label="memory management">
    <div class="memory-head">
      <div>
        <p class="eyebrow">Memory</p>
        <h2>长期记忆管理</h2>
      </div>
      <button class="ghost-button" type="button" :disabled="isLoading" @click="loadMemories">
        {{ isLoading ? "读取中" : "刷新" }}
      </button>
    </div>

    <div class="memory-metrics">
      <span>{{ activeCount }} active</span>
      <span>{{ disabledCount }} disabled</span>
    </div>

    <ul v-if="memories.length" class="memory-list">
      <li v-for="memory in memories" :key="memory.id" :class="{ disabled: memory.status === 'disabled' }">
        <div class="memory-row">
          <span class="status-pill" :class="memory.status">{{ memory.status }}</span>
          <span>{{ memory.type }}</span>
          <strong>{{ formatConfidence(memory.confidence) }}</strong>
        </div>
        <p>{{ memory.content }}</p>
        <small>{{ memory.reason || "no reason" }}</small>
        <div class="memory-actions">
          <button
            v-if="memory.status === 'active'"
            type="button"
            :disabled="updatingId === memory.id"
            @click="setMemoryStatus(memory, 'disabled')"
          >
            禁用
          </button>
          <button
            v-else
            type="button"
            :disabled="updatingId === memory.id"
            @click="setMemoryStatus(memory, 'active')"
          >
            启用
          </button>
        </div>
      </li>
    </ul>

    <p v-else class="muted-row">暂无长期记忆。</p>
    <p v-if="errorMessage" class="error-copy">{{ errorMessage }}</p>
  </section>
</template>

<style scoped>
.memory-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 24px;
  background:
    linear-gradient(135deg, rgba(255, 253, 248, 0.88), rgba(242, 247, 238, 0.8)),
    linear-gradient(90deg, rgba(62, 118, 94, 0.06), transparent);
  box-shadow: 0 16px 34px rgba(43, 76, 88, 0.06);
  backdrop-filter: blur(14px);
}

.memory-head,
.memory-row,
.memory-actions,
.memory-metrics {
  display: flex;
  align-items: center;
  gap: 10px;
}

.memory-head {
  justify-content: space-between;
}

.eyebrow,
h2,
.memory-list,
.memory-list p,
.muted-row,
.error-copy {
  margin: 0;
}

.eyebrow {
  color: #d36f55;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h2 {
  margin-top: 3px;
  color: #20333c;
  font-size: 1.1rem;
  line-height: 1.2;
}

.ghost-button,
.memory-actions button {
  min-height: 34px;
  border: 1px solid rgba(43, 76, 88, 0.14);
  border-radius: 12px;
  padding: 0 11px;
  color: #315f70;
  background: rgba(255, 255, 255, 0.48);
  font-weight: 900;
  cursor: pointer;
}

.ghost-button:disabled,
.memory-actions button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.memory-metrics {
  flex-wrap: wrap;
}

.memory-metrics span,
.status-pill {
  border: 1px solid rgba(43, 76, 88, 0.1);
  border-radius: 999px;
  padding: 6px 8px;
  color: #315f70;
  background: rgba(255, 255, 255, 0.5);
  font-size: 0.78rem;
  font-weight: 900;
}

.status-pill.disabled {
  color: #9a5140;
  border-color: rgba(211, 111, 85, 0.24);
  background: rgba(255, 240, 231, 0.8);
}

.status-pill.active {
  color: #3e765e;
  border-color: rgba(93, 156, 123, 0.24);
  background: rgba(235, 249, 240, 0.82);
}

.memory-list {
  display: grid;
  gap: 8px;
  padding: 0;
  list-style: none;
}

.memory-list li {
  display: grid;
  gap: 7px;
  padding: 10px;
  border: 1px solid rgba(43, 76, 88, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.42);
}

.memory-list li.disabled {
  opacity: 0.72;
}

.memory-row {
  flex-wrap: wrap;
  color: #60717a;
  font-size: 0.8rem;
}

.memory-row strong {
  color: #315f70;
}

.memory-list p {
  color: #294654;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.memory-list small,
.muted-row {
  color: #60717a;
  line-height: 1.45;
}

.memory-actions {
  justify-content: flex-end;
}

.error-copy {
  color: #b33d37;
  font-size: 0.88rem;
  line-height: 1.5;
}

@media (max-width: 720px) {
  .memory-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .ghost-button {
    width: 100%;
  }
}
</style>
