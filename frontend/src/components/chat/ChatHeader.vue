<template>
  <header class="chat-header">
    <div class="title-block">
      <p class="eyebrow">SoulSync / Berry Frontend Senpai</p>
      <h1>前端学姐协作台</h1>
      <p class="summary">
        面向后端开发者的轻量 AI 工作台：对话、Persona、RAG、Memory 和 Trace 都在一屏里确认。
      </p>
    </div>

    <aside class="berry-card" aria-label="berry profile">
      <div class="avatar" aria-hidden="true">
        <span class="avatar-face">Berry</span>
      </div>
      <div>
        <p class="card-label">今日值班</p>
        <p class="card-title">毒舌但靠谱</p>
        <p class="card-copy">页面不会自己变好看，但我会盯着你把链路跑通。</p>
      </div>
    </aside>
  </header>
</template>

<style scoped>
.chat-header {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 22px;
  align-items: stretch;
  padding: clamp(22px, 4vw, 34px);
  overflow: hidden;
  border: 1px solid rgba(43, 76, 88, 0.13);
  border-radius: 32px;
  background:
    linear-gradient(125deg, rgba(255, 253, 248, 0.94), rgba(247, 244, 235, 0.84)),
    radial-gradient(circle at 80% 15%, rgba(255, 202, 180, 0.52), transparent 18rem);
  box-shadow: 0 24px 70px rgba(43, 76, 88, 0.11);
}

.chat-header::after {
  position: absolute;
  right: 28px;
  top: 22px;
  width: 84px;
  height: 18px;
  content: "";
  border-block: 1px solid rgba(221, 120, 91, 0.38);
  transform: rotate(-8deg);
  opacity: 0.9;
}

.title-block {
  display: grid;
  gap: 12px;
  align-content: center;
}

.eyebrow,
h1,
.summary,
.card-label,
.card-title,
.card-copy {
  margin: 0;
}

.eyebrow {
  color: #d36f55;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h1 {
  max-width: 12em;
  color: #20333c;
  font-size: clamp(2.2rem, 6vw, 4.7rem);
  line-height: 0.96;
  letter-spacing: -0.06em;
}

.summary {
  max-width: 44rem;
  color: #52666f;
  font-size: 1rem;
  line-height: 1.75;
}

.berry-card {
  position: relative;
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  gap: 14px;
  align-items: center;
  align-self: end;
  padding: 16px;
  border: 1px solid rgba(43, 76, 88, 0.12);
  border-radius: 26px;
  background: rgba(255, 255, 255, 0.55);
  backdrop-filter: blur(18px);
}

.berry-card::before {
  position: absolute;
  right: 18px;
  bottom: 14px;
  content: "senpai note";
  color: rgba(32, 51, 60, 0.16);
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.avatar {
  display: grid;
  place-items: center;
  width: 86px;
  height: 86px;
  border: 1px solid rgba(221, 120, 91, 0.36);
  border-radius: 30% 70% 62% 38% / 42% 42% 58% 58%;
  background:
    radial-gradient(circle at 50% 38%, #fff9f0 0 8px, transparent 9px),
    linear-gradient(145deg, #ffc9b2, #8bb7ce);
  box-shadow: inset 0 -12px 18px rgba(32, 51, 60, 0.09);
}

.avatar-face {
  color: #20333c;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.card-label {
  color: #d36f55;
  font-size: 0.74rem;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.card-title {
  margin-top: 5px;
  color: #20333c;
  font-size: 1.16rem;
  font-weight: 900;
}

.card-copy {
  margin-top: 6px;
  color: #5a6b73;
  line-height: 1.55;
  font-size: 0.9rem;
}

@media (max-width: 860px) {
  .chat-header {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .berry-card {
    grid-template-columns: 1fr;
  }

  .avatar {
    width: 72px;
    height: 72px;
  }
}
</style>
