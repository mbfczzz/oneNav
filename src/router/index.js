import { createRouter, createWebHistory } from 'vue-router'
import { useNavStore } from '@/stores/nav'

// Lazy-loaded views — mirrors the original HomeView / DashboardView / UserView / NotFoundView chunks.
const routes = [
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
  {
    path: '/dashboard',
    name: 'dashboard',
    meta: { requiresAuth: true },
    component: () => import('@/views/DashboardView.vue'),
  },
  { path: '/user', name: 'user', component: () => import('@/views/UserView.vue') },
  {
    path: '/:pathMatch(.*)*',
    name: 'notFound',
    component: () => import('@/views/NotFoundView.vue'),
  },
]

const router = createRouter({
  history: createWebHistory('/'),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

// Gate the dashboard behind login; bounce to /user with a redirect back.
router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) return
  const store = useNavStore()
  // Validate a persisted token before trusting it (initAuth clears it on 401).
  if (store.token && !store.user) await store.initAuth()
  if (!store.isAuthenticated) {
    return { name: 'user', query: { redirect: to.fullPath } }
  }
})

export default router
