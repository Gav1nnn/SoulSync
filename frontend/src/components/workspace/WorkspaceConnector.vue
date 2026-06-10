<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  connectWorkspace,
  fetchCurrentWorkspace,
  WorkspaceApiError,
} from "../../api/workspace";
import type { Workspace } from "../../types/workspace";

const pathDraft = ref("");
const workspace = ref<Workspace | null>(null);
const isLoading = ref(false);
const errorMessage = ref("");

async function loadCurrentWorkspace() {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    const response = await fetchCurrentWorkspace();
    workspace.value = response.workspace;
    if (response.workspace) {
      pathDraft.value = response.workspace.path;
    }
  } catch (error) {
    errorMessage.value =
      error instanceof WorkspaceApiError ? error.message : "项目状态读取失败。";
  } finally {
    isLoading.value = false;
  }
}

async function submitWorkspace() {
  const nextPath = pathDraft.value.trim();
  if (!nextPath || isLoading.value) {
    return;
  }

  isLoading.value = true;
  errorMessage.value = "";

  try {
    const response = await connectWorkspace(nextPath);
    workspace.value = response.workspace;
    if (response.workspace) {
      pathDraft.value = response.workspace.path;
    }
  } catch (error) {
    errorMessage.value =
      error instanceof WorkspaceApiError ? error.message : "项目连接失败。";
  } finally {
    isLoading.value = false;
  }
}

function formatTime(value: string) {
  if (!value) {
    return "waiting";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

onMounted(() => {
  void loadCurrentWorkspace();
});
</script>

<template>
  <section class="workspace-card" aria-label="workspace connection">
    <div class="workspace-head">
      <div>
        <p class="eyebrow">Workspace</p>
        <h2>连接本地项目</h2>
      </div>
      <button class="ghost-button" type="button" :disabled="isLoading" @click="loadCurrentWorkspace">
        刷新
      </button>
    </div>

    <form class="workspace-form" @submit.prevent="submitWorkspace">
      <label class="path-field">
        <span>绝对路径</span>
        <input
          v-model="pathDraft"
          type="text"
          placeholder="/Users/name/project"
          autocomplete="off"
          spellcheck="false"
        />
      </label>
      <button class="connect-button" type="submit" :disabled="isLoading || !pathDraft.trim()">
        {{ isLoading ? "处理中" : "连接" }}
      </button>
    </form>

    <div v-if="workspace" class="workspace-status">
      <p class="path-value">{{ workspace.path }}</p>
      <div class="status-row">
        <span class="status-pill">{{ workspace.branch }}</span>
        <span class="status-pill" :class="{ dirty: workspace.dirty, clean: !workspace.dirty }">
          {{ workspace.dirty ? "dirty" : "clean" }}
        </span>
        <span class="status-pill muted">{{ formatTime(workspace.updated_at) }}</span>
      </div>
    </div>

    <p v-else class="empty-copy">尚未连接项目。</p>
    <p v-if="errorMessage" class="error-copy">{{ errorMessage }}</p>
  </section>
</template>

<style scoped>
.workspace-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 24px;
  background:
    linear-gradient(135deg, rgba(255, 253, 248, 0.9), rgba(237, 247, 245, 0.82)),
    linear-gradient(90deg, rgba(211, 111, 85, 0.06), transparent);
  box-shadow: 0 16px 34px rgba(43, 76, 88, 0.07);
  backdrop-filter: blur(14px);
}

.workspace-head,
.workspace-form,
.status-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workspace-head {
  justify-content: space-between;
}

.eyebrow,
h2,
.path-field span,
.path-value,
.empty-copy,
.error-copy {
  margin: 0;
}

.eyebrow,
.path-field span {
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

.workspace-form {
  align-items: flex-end;
}

.path-field {
  display: grid;
  flex: 1;
  gap: 7px;
  min-width: 0;
}

.path-field input {
  width: 100%;
  min-height: 42px;
  border: 1px solid rgba(43, 76, 88, 0.14);
  border-radius: 14px;
  padding: 0 12px;
  color: #20333c;
  background: rgba(255, 255, 255, 0.66);
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  font-size: 0.86rem;
  outline: none;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background 160ms ease;
}

.path-field input:focus {
  border-color: rgba(211, 111, 85, 0.46);
  background: #fffdf8;
  box-shadow: 0 0 0 4px rgba(211, 111, 85, 0.1);
}

.connect-button,
.ghost-button {
  min-height: 42px;
  border: 1px solid rgba(43, 76, 88, 0.14);
  border-radius: 14px;
  padding: 0 14px;
  color: #20333c;
  background: #fffdf8;
  font-weight: 900;
  cursor: pointer;
  transition:
    transform 160ms ease,
    border-color 160ms ease,
    opacity 160ms ease;
}

.connect-button {
  color: #fffdf8;
  border-color: rgba(176, 88, 62, 0.5);
  background: #d36f55;
}

.connect-button:hover,
.ghost-button:hover {
  transform: translateY(-1px);
  border-color: rgba(211, 111, 85, 0.46);
}

.connect-button:disabled,
.ghost-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
  transform: none;
}

.workspace-status {
  display: grid;
  gap: 9px;
  padding-top: 2px;
}

.path-value {
  color: #294654;
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  font-size: 0.82rem;
  overflow-wrap: anywhere;
}

.status-row {
  flex-wrap: wrap;
}

.status-pill {
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 999px;
  padding: 6px 9px;
  color: #315f70;
  background: rgba(255, 255, 255, 0.52);
  font-size: 0.78rem;
  font-weight: 900;
}

.status-pill.clean {
  color: #3e765e;
  border-color: rgba(93, 156, 123, 0.24);
  background: rgba(235, 249, 240, 0.82);
}

.status-pill.dirty {
  color: #9a5140;
  border-color: rgba(211, 111, 85, 0.26);
  background: rgba(255, 240, 231, 0.86);
}

.status-pill.muted,
.empty-copy {
  color: #60717a;
}

.empty-copy,
.error-copy {
  font-size: 0.88rem;
  line-height: 1.5;
}

.error-copy {
  color: #b33d37;
}

@media (max-width: 720px) {
  .workspace-head,
  .workspace-form {
    align-items: stretch;
    flex-direction: column;
  }

  .ghost-button,
  .connect-button {
    width: 100%;
  }
}
</style>
