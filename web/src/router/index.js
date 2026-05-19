import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/register', name: 'Register', component: () => import('../views/Register.vue') },
  { path: '/forgot-password', name: 'ForgotPassword', component: () => import('../views/ForgotPassword.vue') },
  { path: '/reset-password', name: 'ResetPassword', component: () => import('../views/ResetPassword.vue') },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { requiresAuth: true } },
  { path: '/domains/:id', name: 'DomainDetail', component: () => import('../views/DomainDetail.vue'), meta: { requiresAuth: true } },
  { path: '/settings', name: 'Settings', component: () => import('../views/Settings.vue'), meta: { requiresAuth: true } },
  { path: '/profile', name: 'Profile', component: () => import('../views/Profile.vue'), meta: { requiresAuth: true } },
  { path: '/permissions', name: 'MyPermissions', component: () => import('../views/MyPermissions.vue'), meta: { requiresAuth: true } },
  { path: '/ram-tokens', name: 'RAMTokens', component: () => import('../views/RAMTokens.vue'), meta: { requiresAuth: true } },
  { path: '/friends', name: 'Friends', component: () => import('../views/Friends.vue'), meta: { requiresAuth: true } },
  { path: '/messages', name: 'Messages', component: () => import('../views/Messages.vue'), meta: { requiresAuth: true } },
  { path: '/messages/:id', name: 'Chat', component: () => import('../views/Chat.vue'), meta: { requiresAuth: true } },
  { path: '/my-logs', name: 'MyLogs', component: () => import('../views/MyLogs.vue'), meta: { requiresAuth: true } },
  { path: '/providers', name: 'Providers', component: () => import('../views/Providers.vue'), meta: { requiresAuth: true } },
  { path: '/admin', name: 'Admin', component: () => import('../views/Admin.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthed) {
    next('/login')
  } else {
    next()
  }
})

export default router
