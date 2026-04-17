import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import { useAuthStore } from '@/stores/auth'
import DefaultLayout from '@/layouts/DefaultLayout.vue'
import BlankLayout from '@/layouts/BlankLayout.vue'
import { useTagsViewStore } from '@/stores/tagsView'
import { cleanupStaleElementOverlays } from '@/utils/overlayCleanup'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: DefaultLayout,
      children: [
        { path: '', name: 'home', component: HomeView, meta: { title: '首页', order: 0, icon: 'House' } },
        { path: 'traffic', name: 'traffic', component: () => import('../views/TrafficView.vue'), meta: { title: '流量监控', permissions: ['traffic.read'], order: 10, icon: 'TrendCharts' } },
        { path: 'schools', name: 'schools', component: () => import('../views/SchoolsView.vue'), meta: { title: '学校管理', permissions: ['school.manage'], order: 20, icon: 'School' } },
        { path: 'settlement/dashboard', redirect: '/settlement/user-query' },
        { path: 'settlement/user-query', name: 'settlement-user-query', component: () => import('../views/SettlementUserQueryView.vue'), meta: { title: '单用户结算查询', permissions: ['settlement.data.read'], order: 33, icon: 'User', group: 'settlement-dashboard' } },
        { path: 'settlement', name: 'settlement', component: () => import('../views/SettlementView.vue'), meta: { title: '结算系统配置', permissions: ['settlement.read', 'settlement.calculate'], order: 30, icon: 'Wallet', group: 'settlement-config' } },
        { path: 'settlement/rates/customer', name: 'settlement-rates-customer', component: () => import('../views/CustomerRatesView.vue'), meta: { title: '客户业务费率', permissions: ['rates.customer.read'], cache: true, order: 31, icon: 'User', group: 'settlement-config' } },
        { path: 'settlement/rates/node', name: 'settlement-rates-node', component: () => import('../views/NodeRatesView.vue'), meta: { title: '节点业务费率', permissions: ['rates.node.read'], cache: true, order: 32, icon: 'Share', group: 'settlement-config' } },
        { path: 'settlement/rates/final', name: 'settlement-rates-final', component: () => import('../views/FinalCustomerRatesView.vue'), meta: { title: '最终客户费率', permissions: ['rates.final.read'], cache: true, order: 33, icon: 'CircleCheck', group: 'settlement-config' } },
        { path: 'settlement/rates/discount-rules', name: 'settlement-rates-discount-rules', component: () => import('../views/DiscountRulesView.vue'), meta: { title: '折损规则管理', permissions: ['rates.discount_rule.read'], order: 38, icon: 'Coin', group: 'settlement-config' } },
        { path: 'settlement/channels', redirect: '/settlement/user-query' },
        { path: 'settlement/rates/filter-rules', name: 'settlement-rates-filter-rules', component: () => import('../views/FilterRulesView.vue'), meta: { title: '过滤规则管理', permissions: ['rates.filter_rules.read'], order: 39, icon: 'Filter', group: 'settlement-config' } },
        { path: 'settlement/rates/sync-rules', name: 'settlement-rates-sync-rules', component: () => import('../views/SyncRulesView.vue'), meta: { title: '同步规则管理', permissions: ['rates.sync_rules.read'], order: 40, icon: 'Refresh', group: 'settlement-config' } },
        { path: 'settlement/business-types', name: 'settlement-business-types', component: () => import('../views/BusinessTypesView.vue'), meta: { title: '业务类型管理', permissions: ['business_types.read'], cache: true, order: 36, icon: 'Box', group: 'settlement-config', hideInMenu: true } },
        { path: 'operation-logs', name: 'operation-logs', component: () => import('../views/OperationLogsView.vue'), meta: { title: '操作日志', permissions: ['operation_logs.read'], cache: true, order: 70, icon: 'Notebook' } },
        { path: 'system/users', name: 'system-users', component: () => import('../views/SystemUsersView.vue'), meta: { title: '用户管理', permissions: ['system.user.manage'], order: 80, icon: 'UserFilled' } },
        { path: 'system/roles', name: 'system-roles', component: () => import('../views/SystemRolesView.vue'), meta: { title: '角色管理', permissions: ['system.role.manage'], order: 81, icon: 'Avatar' } },
        { path: 'system/permissions', name: 'system-permissions', component: () => import('../views/SystemPermissionsView.vue'), meta: { title: '权限设置', permissions: ['system.role.manage', 'system.permission.manage'], order: 82, icon: 'Lock' } },
        { path: 'system/traffic-scopes', name: 'system-traffic-scopes', component: () => import('../views/SystemTrafficScopesView.vue'), meta: { title: '流量范围管理', permissions: ['traffic.scope.manage'], order: 83, icon: 'Connection' } },
        { path: 'system/settings', name: 'system-settings', component: () => import('../views/SystemSettingsView.vue'), meta: { title: '系统设置', permissions: ['system.user.manage'], order: 84, icon: 'Setting' } },
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

function isPublicMeta(meta: unknown): boolean {
  if (!meta || typeof meta !== 'object') return false
  return Boolean((meta as { public?: boolean }).public)
}

// 鉴权与权限守卫 + 设置页面标题
router.beforeEach(async (to, from, next) => {
  document.title = `${to.meta.title || '学校流量监控系统'} - NFA Dashboard`
  const auth = useAuthStore()
  if (!auth.token) auth.initFromStorage()

  // 公共路由放行
  if (isPublicMeta(to.meta)) return next()

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
    if (!isPublicMeta(to.meta)) {
      tags.addRoute(to)
    }
  } catch {}
  try {
    window.setTimeout(() => {
      cleanupStaleElementOverlays(document)
    }, 0)
  } catch {}
})

export default router
