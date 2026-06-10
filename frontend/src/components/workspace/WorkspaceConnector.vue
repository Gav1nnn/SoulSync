<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  connectWorkspace,
  fetchCurrentWorkspace,
  fetchCurrentWorkspaceSummary,
  WorkspaceApiError,
} from "../../api/workspace";
import type { Workspace, WorkspaceCandidate, WorkspaceSummary } from "../../types/workspace";

const pathDraft = ref("");
const workspace = ref<Workspace | null>(null);
const summary = ref<WorkspaceSummary | null>(null);
const isLoading = ref(false);
const isSummaryLoading = ref(false);
const errorMessage = ref("");
const summaryError = ref("");

async function loadCurrentWorkspace() {
  isLoading.value = true;
  errorMessage.value = "";
  summaryError.value = "";

  try {
    const response = await fetchCurrentWorkspace();
    workspace.value = response.workspace;
    if (response.workspace) {
      pathDraft.value = response.workspace.path;
      await loadSummary();
    } else {
      summary.value = null;
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
  summaryError.value = "";

  try {
    const response = await connectWorkspace(nextPath);
    workspace.value = response.workspace;
    if (response.workspace) {
      pathDraft.value = response.workspace.path;
      await loadSummary();
    }
  } catch (error) {
    errorMessage.value =
      error instanceof WorkspaceApiError ? error.message : "项目连接失败。";
  } finally {
    isLoading.value = false;
  }
}

async function loadSummary() {
  if (!workspace.value || isSummaryLoading.value) {
    return;
  }

  isSummaryLoading.value = true;
  summaryError.value = "";

  try {
    const response = await fetchCurrentWorkspaceSummary();
    summary.value = response.summary;
  } catch (error) {
    summary.value = null;
    summaryError.value =
      error instanceof WorkspaceApiError ? error.message : "项目摘要读取失败。";
  } finally {
    isSummaryLoading.value = false;
  }
}

function formatList(values: string[]) {
  return values.length ? values.join(" / ") : "none";
}

function limitedCandidates(candidates: WorkspaceCandidate[]) {
  return candidates.slice(0, 5);
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
  <div class="workspace-stack">
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

    <section class="summary-card" aria-label="workspace summary">
      <div class="workspace-head">
        <div>
          <p class="eyebrow">Summary</p>
          <h2>项目状态卡片</h2>
        </div>
        <button class="ghost-button" type="button" :disabled="!workspace || isSummaryLoading" @click="loadSummary">
          {{ isSummaryLoading ? "扫描中" : "扫描" }}
        </button>
      </div>

      <template v-if="summary">
        <div class="summary-metrics">
          <div>
            <span>Package</span>
            <strong>{{ formatList(summary.package_managers) }}</strong>
          </div>
          <div>
            <span>Frontend</span>
            <strong>{{ formatList(summary.frontend_frameworks) }}</strong>
          </div>
          <div>
            <span>Backend</span>
            <strong>{{ formatList(summary.backend_frameworks) }}</strong>
          </div>
        </div>

        <div class="summary-grid">
          <section class="summary-section">
            <p class="section-title">Tree</p>
            <ul class="compact-list">
              <li v-for="item in summary.tree.slice(0, 8)" :key="item.path">
                <span>{{ item.type }}</span>
                <code>{{ item.path }}</code>
              </li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">Routes</p>
            <ul class="compact-list">
              <li v-for="item in limitedCandidates(summary.backend_route_candidates)" :key="item.path">
                <span>{{ item.kind }}</span>
                <code>{{ item.path }}</code>
              </li>
              <li v-if="!summary.backend_route_candidates.length" class="muted-row">none</li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">API</p>
            <ul class="compact-list">
              <li v-for="item in summary.api_candidates.slice(0, 5)" :key="`${item.method}-${item.path}-${item.handler}`">
                <span>{{ item.method }}</span>
                <code>{{ item.path }}</code>
                <small>{{ item.handler || "inline" }} · {{ item.handler_file }}</small>
              </li>
              <li v-if="!summary.api_candidates.length" class="muted-row">none</li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">Frontend</p>
            <ul class="compact-list">
              <li v-for="item in limitedCandidates(summary.frontend_entry_candidates)" :key="item.path">
                <span>{{ item.kind }}</span>
                <code>{{ item.path }}</code>
              </li>
              <li v-if="!summary.frontend_entry_candidates.length" class="muted-row">none</li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">API Client</p>
            <ul class="compact-list">
              <li v-for="item in limitedCandidates(summary.api_client_candidates)" :key="item.path">
                <span>{{ item.kind }}</span>
                <code>{{ item.path }}</code>
              </li>
              <li v-if="!summary.api_client_candidates.length" class="muted-row">none</li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">Types</p>
            <ul class="compact-list">
              <li v-for="item in limitedCandidates(summary.type_file_candidates)" :key="item.path">
                <span>{{ item.kind }}</span>
                <code>{{ item.path }}</code>
              </li>
              <li v-if="!summary.type_file_candidates.length" class="muted-row">none</li>
            </ul>
          </section>

          <section class="summary-section">
            <p class="section-title">Verify</p>
            <ul class="compact-list">
              <li v-for="command in summary.validation_commands.slice(0, 5)" :key="command">
                <span>cmd</span>
                <code>{{ command }}</code>
              </li>
              <li v-if="!summary.validation_commands.length" class="muted-row">none</li>
            </ul>
          </section>
        </div>
      </template>

      <p v-else class="empty-copy">等待项目摘要。</p>
      <p v-if="summaryError" class="error-copy">{{ summaryError }}</p>
    </section>
  </div>
</template>

<style scoped>
.workspace-stack {
  display: grid;
  gap: 14px;
}

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

.summary-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 24px;
  background:
    linear-gradient(135deg, rgba(255, 253, 248, 0.86), rgba(248, 242, 232, 0.82)),
    repeating-linear-gradient(
      135deg,
      rgba(43, 76, 88, 0.035) 0,
      rgba(43, 76, 88, 0.035) 1px,
      transparent 1px,
      transparent 11px
    );
  box-shadow: 0 16px 34px rgba(43, 76, 88, 0.06);
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

.summary-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
}

.summary-metrics div {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 10px;
  border: 1px solid rgba(43, 76, 88, 0.1);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.48);
}

.summary-metrics span,
.section-title,
.compact-list span {
  color: #d36f55;
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.summary-metrics strong {
  min-width: 0;
  color: #20333c;
  font-size: 0.9rem;
  overflow-wrap: anywhere;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.summary-section {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.section-title {
  margin: 0;
  color: #315f70;
}

.compact-list {
  display: grid;
  gap: 6px;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.compact-list li {
  display: grid;
  grid-template-columns: minmax(70px, 0.36fr) minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  min-width: 0;
  padding: 8px;
  border: 1px solid rgba(43, 76, 88, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.42);
}

.compact-list code {
  min-width: 0;
  color: #294654;
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  font-size: 0.76rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.compact-list .muted-row {
  display: block;
  color: #60717a;
  font-size: 0.84rem;
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

  .summary-metrics,
  .summary-grid {
    grid-template-columns: 1fr;
  }

  .compact-list li {
    grid-template-columns: 1fr;
  }
}
</style>
