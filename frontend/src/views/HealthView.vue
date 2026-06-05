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
  color: #20333c;
  background:
    radial-gradient(circle at 20% 20%, rgba(255, 202, 180, 0.56), transparent 22rem),
    linear-gradient(135deg, #fff8ef, #edf4f1);
  font-family:
    "Tsukimi Rounded",
    "Yuanti SC",
    "Hiragino Maru Gothic ProN",
    "PingFang SC",
    sans-serif;
  text-align: center;
}

.health-card {
  display: grid;
  gap: 0.75rem;
  padding: 32px;
  border: 1px solid rgba(43, 76, 88, 0.13);
  border-radius: 28px;
  background: rgba(255, 253, 248, 0.84);
  box-shadow: 0 24px 70px rgba(43, 76, 88, 0.11);
}

h1,
p,
.eyebrow {
  margin: 0;
}

h1 {
  font-size: clamp(2rem, 8vw, 4rem);
  letter-spacing: -0.06em;
}

p {
  color: #60717a;
}

.eyebrow {
  color: #d36f55;
  font-size: 0.76rem;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
</style>
