import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import { useAuthStore } from '@/stores/auth'
import DefaultLayout from '@/layouts/DefaultLayout.vue'
import BlankLayout from '@/layouts/BlankLayout.vue'
import { useTagsViewStore } from '@/stores/tagsView'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: DefaultLayout,
      children: [
        { path: '', name: 'home', component: HomeView, meta: { title: '首页', order: 0, icon: '🏠' } },
        { path: 'traffic', name: 'traffic', component: () => import('../views/TrafficView.vue'), meta: { title: '流量监控', permissions: ['traffic.read'], order: 10, icon: '📈' } },
        { path: 'schools', name: 'schools', component: () => import('../views/SchoolsView.vue'), meta: { title: '学校管理', permissions: ['school.manage'], order: 20, icon: '🏫' } },
        { path: 'settlement/dashboard', name: 'settlement-dashboard', component: () => import('../views/SettlementDashboardView.vue'), meta: { title: '结算结果总览', permissions: ['settlement.results.read'], order: 29, icon: '📊', group: 'settlement-dashboard' } },
        { path: 'settlement', name: 'settlement', component: () => import('../views/SettlementView.vue'), meta: { title: '结算系统配置', permissions: ['settlement.read', 'settlement.calculate'], order: 30, icon: '💰', group: 'settlement-config' } },
        { path: 'settlement/rates/customer', name: 'settlement-rates-customer', component: () => import('../views/CustomerRatesView.vue'), meta: { title: '客户业务费率', permissions: ['rates.customer.read'], cache: true, order: 31, icon: '👤', group: 'settlement-config' } },
        { path: 'settlement/rates/node', name: 'settlement-rates-node', component: () => import('../views/NodeRatesView.vue'), meta: { title: '节点业务费率', permissions: ['rates.node.read'], cache: true, order: 32, icon: '🕸️', group: 'settlement-config' } },
        { path: 'settlement/rates/final', name: 'settlement-rates-final', component: () => import('../views/FinalCustomerRatesView.vue'), meta: { title: '最终客户费率', permissions: ['rates.final.read'], cache: true, order: 33, icon: '✅', group: 'settlement-config' } },
        { path: 'settlement/rates/sync-rules', name: 'settlement-rates-sync-rules', component: () => import('../views/SyncRulesView.vue'), meta: { title: '同步规则管理', permissions: ['rates.sync_rules.read'], order: 39, icon: '🔄', group: 'settlement-config' } },
        { path: 'settlement/entities', name: 'settlement-entities', component: () => import('../views/SettlementEntitiesView.vue'), meta: { title: '业务对象', permissions: ['entities.read'], cache: true, order: 35, icon: '🧩', group: 'settlement-config' } },
        { path: 'settlement/business-types', name: 'settlement-business-types', component: () => import('../views/BusinessTypesView.vue'), meta: { title: '业务类型管理', permissions: ['business_types.read'], cache: true, order: 36, icon: '📦', group: 'settlement-config' } },
        { path: 'operation-logs', name: 'operation-logs', component: () => import('../views/OperationLogsView.vue'), meta: { title: '操作日志', permissions: ['operation_logs.read'], cache: true, order: 70, icon: '📝' } },
        { path: 'system/users', name: 'system-users', component: () => import('../views/SystemUsersView.vue'), meta: { title: '用户管理', permissions: ['system.user.manage'], order: 80, icon: '👥' } },
        { path: 'system/roles', name: 'system-roles', component: () => import('../views/SystemRolesView.vue'), meta: { title: '角色管理', permissions: ['system.role.manage'], order: 81, icon: '🧩' } },
        { path: 'system/permissions', name: 'system-permissions', component: () => import('../views/SystemPermissionsView.vue'), meta: { title: '权限设置', permissions: ['system.role.manage', 'system.permission.manage'], order: 82, icon: '🔐' } },
      ]
    },
    {
      path: '/',
      component: BlankLayout,
      children: [
        { path: 'login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { title: '登录', public: true } },
        { path: '403', name: 'forbidden', component: () => import('../views/ForbiddenView.vue'), meta: { title: '无权限', public: true } },
      ]
    },
    { path: '/:pathMatch(.*)*', redirect: '/' }
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

// 路由后置：收集多页签标签
router.afterEach((to) => {
  try {
    const tags = useTagsViewStore()
    // public 页面（如 login/403）不纳入标签
    if (!(to.meta && (to.meta as any).public)) {
      tags.addRoute(to)
    }
  } catch {}
})

export default router
