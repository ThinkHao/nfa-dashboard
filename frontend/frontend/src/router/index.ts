import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'

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
        title: '流量监控'
      }
    },
    {
      path: '/schools',
      name: 'schools',
      component: () => import('../views/SchoolsView.vue'),
      meta: {
        title: '学校管理'
      }
    },
    {
      path: '/settlement',
      name: 'settlement',
      component: () => import('../views/SettlementView.vue'),
      meta: {
        title: '结算系统'
      }
    }
  ],
})

// 设置页面标题
router.beforeEach((to, from, next) => {
  document.title = `${to.meta.title || '学校流量监控系统'} - NFA Dashboard`
  next()
})

export default router
