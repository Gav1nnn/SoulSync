<template>
  <main class="health-page">
    <section class="health-card">
      <p class="eyebrow">SoulSync frontend</p>
      <h1>frontend ok</h1>
      <p>SoulSync frontend health check is running.</p>
    </section>
  </main>
</template>

<style scoped>
.health-page {
  min-height: 100vh;
  display: grid;
  place-content: center;
  margin: 0;
  padding: 20px;
  color: #17252d;
  background:
    linear-gradient(90deg, rgba(23, 37, 45, 0.055) 1px, transparent 1px),
    linear-gradient(rgba(23, 37, 45, 0.045) 1px, transparent 1px),
    linear-gradient(135deg, #f4efe4, #eef3ef);
  background-size: 28px 28px, 28px 28px, auto;
  font-family: "Avenir Next", "PingFang SC", "Hiragino Sans GB", sans-serif;
  text-align: center;
}

.health-card {
  display: grid;
  gap: 0.75rem;
  min-width: min(420px, calc(100vw - 40px));
  padding: 24px;
  border: 1px solid rgba(43, 76, 88, 0.13);
  border-radius: 8px;
  background: rgba(255, 252, 244, 0.92);
  box-shadow: 0 18px 48px rgba(34, 52, 60, 0.08);
}

h1,
p,
.eyebrow {
  margin: 0;
}

h1 {
  font-family: "Avenir Next Condensed", "DIN Condensed", "PingFang SC", sans-serif;
  font-size: clamp(1.8rem, 7vw, 3rem);
  letter-spacing: 0;
}

p {
  color: #657780;
}

.eyebrow {
  color: #c85f49;
  font-size: 0.76rem;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
</style>
