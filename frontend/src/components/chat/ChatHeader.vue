<template>
  <header class="chat-header">
    <p class="eyebrow">SoulSync / Berry</p>
    <h1>协作对话台</h1>
    <p class="summary">
      这一版先把产品外壳稳定下来：输入问题、查看回复、确认状态、预留 trace 和上下文展示位。
    </p>
  </header>
</template>

<style scoped>
.chat-header {
  display: grid;
  gap: 10px;
  padding: 24px;
  border: 1px solid rgba(126, 67, 81, 0.14);
  border-radius: 24px;
  background: #fffaf8;
}

.eyebrow,
h1,
.summary {
  margin: 0;
}

.eyebrow {
  color: #8f5864;
  font-size: 0.8rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

h1 {
  font-size: clamp(1.8rem, 4vw, 2.8rem);
  line-height: 1.05;
}

.summary {
  max-width: 48rem;
  color: #5f474d;
  line-height: 1.6;
}
</style>
