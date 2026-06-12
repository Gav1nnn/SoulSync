<template>
  <header class="chat-header">
    <div class="title-block">
      <p class="eyebrow">SoulSync</p>
      <h1>Frontend Agent Console</h1>
      <p class="summary">Berry 接管前端补位：连接项目、生成页面、验证分支。</p>
    </div>

    <aside class="berry-card" aria-label="berry profile">
      <div>
        <p class="card-label">Operator</p>
        <p class="card-title">Berry</p>
      </div>
      <span class="live-mark">online</span>
    </aside>
  </header>
</template>

<style scoped>
.chat-header {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 18px;
  align-items: center;
  padding: 18px 20px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background: var(--panel-strong);
  box-shadow: var(--shadow-soft);
}

.title-block {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.eyebrow,
h1,
.summary,
.card-label,
.card-title {
  margin: 0;
}

.eyebrow {
  color: var(--accent);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h1 {
  color: var(--ink);
  font-family: var(--font-display);
  font-size: clamp(1.35rem, 3vw, 2.1rem);
  line-height: 1;
  letter-spacing: 0;
}

.summary {
  max-width: 52rem;
  color: var(--muted);
  font-size: 0.92rem;
  line-height: 1.45;
}

.berry-card {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  min-width: 220px;
  padding: 10px 12px;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  background: var(--panel);
}

.card-label {
  color: var(--muted);
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.card-title {
  margin-top: 3px;
  color: var(--ink);
  font-size: 1rem;
  font-weight: 900;
}

.live-mark {
  border: 1px solid rgba(55, 126, 91, 0.32);
  border-radius: 999px;
  padding: 5px 8px;
  color: var(--ok);
  background: rgba(230, 247, 238, 0.92);
  font-size: 0.74rem;
  font-weight: 900;
}

@media (max-width: 860px) {
  .chat-header {
    grid-template-columns: 1fr;
  }

  .berry-card {
    min-width: 0;
  }
}
</style>
