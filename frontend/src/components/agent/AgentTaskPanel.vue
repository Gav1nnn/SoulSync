<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import { AgentApiError, createAgentTask, fetchAgentTask } from "../../api/agent";
import type { AgentTask, AgentTaskStatus } from "../../types/agent";

const goalDraft = ref("根据用户列表接口生成前端页面");
const task = ref<AgentTask | null>(null);
const isSubmitting = ref(false);
const errorMessage = ref("");
let pollTimer: number | null = null;

const statusLabels: Record<AgentTaskStatus, string> = {
  queued: "排队中",
  planning: "规划中",
  running: "运行中",
  verifying: "验证中",
  completed: "已完成",
  failed: "失败",
};

const isActiveTask = computed(() =>
  task.value
    ? !["completed", "failed"].includes(task.value.status)
    : false,
);

async function submitTask() {
  const goal = goalDraft.value.trim();
  if (!goal || isSubmitting.value) {
    return;
  }

  stopPolling();
  isSubmitting.value = true;
  errorMessage.value = "";

  try {
    const response = await createAgentTask(goal);
    task.value = response.task;
    startPolling(response.task.id);
  } catch (error) {
    errorMessage.value =
      error instanceof AgentApiError ? error.message : "任务创建失败。";
  } finally {
    isSubmitting.value = false;
  }
}

function startPolling(taskId: string) {
  stopPolling();
  pollTimer = window.setInterval(() => {
    void refreshTask(taskId);
  }, 900);
  void refreshTask(taskId);
}

function stopPolling() {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer);
    pollTimer = null;
  }
}

async function refreshTask(taskId = task.value?.id) {
  if (!taskId) {
    return;
  }

  try {
    const response = await fetchAgentTask(taskId);
    task.value = response.task;
    if (response.task.status === "completed" || response.task.status === "failed") {
      stopPolling();
    }
  } catch (error) {
    errorMessage.value =
      error instanceof AgentApiError ? error.message : "任务读取失败。";
    stopPolling();
  }
}

function formatTime(value: string) {
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

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <section class="agent-card" aria-label="agent task panel">
    <div class="agent-head">
      <div>
        <p class="eyebrow">Agent Task</p>
        <h2>Agent 任务运行</h2>
      </div>
      <span v-if="task" class="status-pill" :class="`status-${task.status}`">
        {{ statusLabels[task.status] }}
      </span>
    </div>

    <form class="task-form" @submit.prevent="submitTask">
      <label class="goal-field">
        <span>目标</span>
        <textarea v-model="goalDraft" rows="3" />
      </label>
      <button class="start-button" type="submit" :disabled="isSubmitting || !goalDraft.trim()">
        {{ isSubmitting ? "创建中" : "创建任务" }}
      </button>
    </form>

    <template v-if="task">
      <div class="task-meta">
        <code>{{ task.id }}</code>
        <code v-if="task.branch_name">{{ task.branch_name }}</code>
        <span v-if="task.planner">{{ task.planner }}</span>
        <span>{{ formatTime(task.updated_at) }}</span>
        <span v-if="isActiveTask">polling</span>
      </div>

      <section class="task-section">
        <p class="section-title">Plan</p>
        <ol class="plan-list">
          <li v-for="step in task.plan" :key="step">{{ step }}</li>
          <li v-if="!task.plan.length" class="muted-row">等待 mock planner 输出。</li>
        </ol>
      </section>

      <section class="task-section">
        <p class="section-title">Planner Context</p>
        <div class="tag-list" v-if="task.planner_context_used.length">
          <span v-for="item in task.planner_context_used" :key="item">{{ item }}</span>
        </div>
        <p v-else class="muted-row">等待 planner 上下文。</p>
      </section>

      <section class="task-section">
        <p class="section-title">Read Queue</p>
        <ul class="file-list">
          <li v-for="file in task.files_to_read" :key="file">
            <code>{{ file }}</code>
          </li>
          <li v-if="!task.files_to_read.length" class="muted-row">planner 暂未指定读取文件。</li>
        </ul>
      </section>

      <section class="task-section" v-if="task.initial_action">
        <p class="section-title">Initial Action</p>
        <div class="action-box">
          <span>{{ task.initial_action.type }}</span>
          <code v-if="task.initial_action.path">{{ task.initial_action.path }}</code>
          <code v-if="task.initial_action.query">{{ task.initial_action.query }}</code>
          <p>{{ task.initial_action.reason }}</p>
        </div>
      </section>

      <section class="task-section">
        <p class="section-title">Logs</p>
        <ul class="log-list">
          <li v-for="log in task.logs" :key="`${log.at}-${log.message}`">
            <span>{{ formatTime(log.at) }}</span>
            <strong>{{ statusLabels[log.status] }}</strong>
            <p>{{ log.message }}</p>
          </li>
        </ul>
      </section>

      <section class="task-section">
        <p class="section-title">Changed Files</p>
        <ul class="file-list">
          <li v-for="file in task.changed_files" :key="file">
            <code>{{ file }}</code>
          </li>
          <li v-if="!task.changed_files.length" class="muted-row">等待写入阶段。</li>
        </ul>
      </section>

      <section class="task-section">
        <p class="section-title">Verification</p>
        <div v-if="task.verification" class="verification-box">
          <span :class="`verify-${task.verification.status}`">{{ task.verification.status }}</span>
          <code>{{ task.verification.command }}</code>
          <p v-for="line in task.verification.output" :key="line">{{ line }}</p>
        </div>
        <p v-else class="muted-row">等待验证阶段。</p>
      </section>

      <p v-if="task.error" class="error-copy">{{ task.error }}</p>
    </template>

    <p v-else class="muted-row">先连接 workspace，再创建 agent task。</p>
    <p v-if="errorMessage" class="error-copy">{{ errorMessage }}</p>
  </section>
</template>

<style scoped>
.agent-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 24px;
  background:
    linear-gradient(135deg, rgba(255, 253, 248, 0.88), rgba(238, 247, 245, 0.8)),
    linear-gradient(90deg, rgba(49, 95, 112, 0.06), transparent);
  box-shadow: 0 16px 34px rgba(43, 76, 88, 0.06);
  backdrop-filter: blur(14px);
}

.agent-head,
.task-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.eyebrow,
h2,
.goal-field span,
.section-title,
.log-list p,
.error-copy,
.muted-row {
  margin: 0;
}

.eyebrow,
.goal-field span,
.section-title {
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

.status-pill {
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 999px;
  padding: 6px 9px;
  color: #315f70;
  background: rgba(255, 255, 255, 0.54);
  font-size: 0.78rem;
  font-weight: 900;
}

.status-completed {
  color: #3e765e;
  border-color: rgba(93, 156, 123, 0.24);
  background: rgba(235, 249, 240, 0.82);
}

.status-failed {
  color: #b33d37;
  border-color: rgba(179, 61, 55, 0.24);
  background: rgba(255, 236, 232, 0.82);
}

.status-planning,
.status-running,
.status-verifying {
  color: #9a5140;
  border-color: rgba(211, 111, 85, 0.26);
  background: rgba(255, 240, 231, 0.86);
}

.task-form {
  display: grid;
  gap: 10px;
}

.goal-field {
  display: grid;
  gap: 7px;
}

.goal-field textarea {
  width: 100%;
  min-height: 86px;
  resize: vertical;
  border: 1px solid rgba(43, 76, 88, 0.14);
  border-radius: 16px;
  padding: 11px 12px;
  color: #20333c;
  background: rgba(255, 255, 255, 0.66);
  line-height: 1.5;
  outline: none;
}

.goal-field textarea:focus {
  border-color: rgba(211, 111, 85, 0.46);
  background: #fffdf8;
  box-shadow: 0 0 0 4px rgba(211, 111, 85, 0.1);
}

.start-button {
  justify-self: start;
  min-height: 42px;
  border: 1px solid rgba(176, 88, 62, 0.5);
  border-radius: 14px;
  padding: 0 14px;
  color: #fffdf8;
  background: #d36f55;
  font-weight: 900;
  cursor: pointer;
}

.start-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.task-meta {
  flex-wrap: wrap;
  justify-content: flex-start;
  color: #60717a;
  font-size: 0.82rem;
}

.task-meta code,
.verification-box code {
  color: #294654;
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  overflow-wrap: anywhere;
}

.task-section {
  display: grid;
  gap: 8px;
}

.plan-list,
.log-list,
.file-list {
  display: grid;
  gap: 7px;
  margin: 0;
  padding-left: 18px;
}

.plan-list li,
.log-list li,
.file-list li,
.verification-box {
  color: #294654;
  line-height: 1.45;
  font-size: 0.88rem;
}

.log-list,
.file-list {
  padding-left: 0;
  list-style: none;
}

.file-list li {
  padding: 8px;
  border: 1px solid rgba(43, 76, 88, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.42);
}

.file-list code {
  color: #294654;
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  overflow-wrap: anywhere;
}

.log-list li {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  padding: 8px;
  border: 1px solid rgba(43, 76, 88, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.42);
}

.log-list span {
  color: #60717a;
  font-size: 0.78rem;
}

.log-list strong {
  color: #315f70;
  font-size: 0.78rem;
}

.verification-box,
.action-box {
  display: grid;
  gap: 5px;
  padding: 10px;
  border: 1px solid rgba(43, 76, 88, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.42);
}

.verification-box span,
.action-box span {
  color: #3e765e;
  font-size: 0.78rem;
  font-weight: 900;
  text-transform: uppercase;
}

.verification-box .verify-failed {
  color: #b33d37;
}

.verification-box .verify-skipped {
  color: #9a5140;
}

.verification-box p,
.action-box p {
  margin: 0;
  color: #60717a;
}

.action-box code {
  color: #294654;
  font-family: "Menlo", "Monaco", "Courier New", monospace;
  overflow-wrap: anywhere;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.tag-list span {
  border: 1px solid rgba(43, 76, 88, 0.1);
  border-radius: 999px;
  padding: 7px 9px;
  color: #315f70;
  background: rgba(255, 255, 255, 0.48);
  font-size: 0.78rem;
  font-weight: 800;
}

.muted-row {
  color: #60717a;
  font-size: 0.88rem;
  line-height: 1.5;
}

.error-copy {
  color: #b33d37;
  font-size: 0.88rem;
  line-height: 1.5;
}

@media (max-width: 720px) {
  .agent-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .start-button {
    width: 100%;
  }

  .log-list li {
    grid-template-columns: 1fr;
  }
}
</style>
