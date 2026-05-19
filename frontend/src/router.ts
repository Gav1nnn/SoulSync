import { createRouter, createWebHistory } from "vue-router";
import ChatView from "./views/ChatView.vue";
import HealthView from "./views/HealthView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "chat",
      component: ChatView,
    },
    {
      path: "/health",
      name: "health",
      component: HealthView,
    },
  ],
});

export default router;
