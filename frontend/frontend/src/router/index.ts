import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
      meta: {
        title: '首页'
      }
    },
    {
      path: '/traffic',
      name: 'traffic',
      component: () => import('../views/TrafficView.vue'),
      meta: {
        title: '流量监控',
        permissions: ['traffic.read']
      }
    },
    {
      path: '/schools',
      name: 'schools',
      component: () => import('../views/SchoolsView.vue'),
      meta: {
        title: '学校管理',
        permissions: ['school.manage']
      }
    },
    {
      path: '/settlement',
      name: 'settlement',
      component: () => import('../views/SettlementView.vue'),
      meta: {
        title: '结算系统',
        permissions: ['settlement.read', 'settlement.calculate']
      }
    },
    {
      path: '/operation-logs',
      name: 'operation-logs',
      component: () => import('../views/OperationLogsView.vue'),
      meta: {
        title: '操作日志',
        permissions: ['operation_logs.read']
      }
    },
    {
      path: '/system/users',
      name: 'system-users',
      component: () => import('../views/SystemUsersView.vue'),
      meta: {
        title: '用户管理',
        permissions: ['system.user.manage']
      }
    },
    {
      path: '/system/roles',
      name: 'system-roles',
      component: () => import('../views/SystemRolesView.vue'),
      meta: {
        title: '角色管理',
        permissions: ['system.role.manage']
      }
    },
    {
      path: '/system/permissions',
      name: 'system-permissions',
      component: () => import('../views/SystemPermissionsView.vue'),
      meta: {
        title: '权限设置',
        permissions: ['system.role.manage', 'system.permission.manage']
      }
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { title: '登录', public: true }
    },
    {
      path: '/403',
      name: 'forbidden',
      component: () => import('../views/ForbiddenView.vue'),
      meta: { title: '无权限', public: true }
    }
  ],
})

// 鉴权与权限守卫 + 设置页面标题
router.beforeEach(async (to, from, next) => {
  document.title = `${to.meta.title || '学校流量监控系统'} - NFA Dashboard`
  const auth = useAuthStore()
  if (!auth.token) auth.initFromStorage()

  // 公共路由放行
  if (to.meta && (to.meta as any).public) return next()

  // 未登录，跳转登录
  if (!auth.token) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }

  // 根据需要加载一次 Profile（避免页面刷新后权限丢失）
  if ((!auth.user || !auth.permissions?.length) && !auth.loadingProfile) {
    try { await auth.loadProfile() } catch {}
  }

  // 权限校验
  const required = (to.meta?.permissions as string[] | undefined) || []
  if (required.length && !auth.hasAnyPermission(required)) {
    return next('/403')
  }
  next()
})

export default router
