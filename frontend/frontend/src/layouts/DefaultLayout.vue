<script setup lang="ts">
import { RouterView } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import Breadcrumb from '@/components/Breadcrumb/index.vue'
import TagsView from '@/components/TagsView/index.vue'
import SidebarMenu from '@/components/Sidebar/Menu.vue'
import Navbar from '@/components/Navbar/index.vue'

const auth = useAuthStore()

const isAuthed = computed(() => auth.isAuthenticated)
const canOpLogs = computed(() => auth.hasPermission('operation_logs.read'))
const canSettlement = computed(() => auth.hasAnyPermission(['settlement.read','settlement.calculate']))
const canTraffic = computed(() => auth.hasPermission('traffic.read'))
const canSchools = computed(() => auth.hasPermission('school.manage'))
const canSysUser = computed(() => auth.hasPermission('system.user.manage'))
const canSysRole = computed(() => auth.hasPermission('system.role.manage'))
const canSysPerm = computed(() => auth.hasAnyPermission(['system.role.manage', 'system.permission.manage']))
const canRatesCustomer = computed(() => auth.hasPermission('rates.customer.read'))
const canRatesNode = computed(() => auth.hasPermission('rates.node.read'))
const canRatesFinal = computed(() => auth.hasPermission('rates.final.read'))
const canEntities = computed(() => auth.hasPermission('entities.read'))
const canBusinessTypes = computed(() => auth.hasPermission('business_types.read'))

// 顶栏相关逻辑已移入 Navbar 组件

// settings panel 使用独立组件 SettingsDrawer
</script>

<template>
  <el-container class="layout">
    <el-aside width="220px" class="sidebar glass-sidebar">
      <div class="logo">
        <span class="brand">NFA Dashboard</span>
      </div>
      <SidebarMenu />
    </el-aside>

    <el-container>
      <Navbar />

      <div class="breadcrumb-bar">
        <Breadcrumb />
      </div>

      <TagsView />

      <el-main class="content">
        <RouterView v-slot="{ Component, route }">
          <template v-if="route.meta && (route.meta as any).cache">
            <keep-alive>
              <component :is="Component" />
            </keep-alive>
          </template>
          <template v-else>
            <component :is="Component" />
          </template>
        </RouterView>
      </el-main>
      <el-footer class="app-footer">
        <p> 2025 学校流量监控系统 - NFA Dashboard</p>
      </el-footer>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout { min-height: 100vh; background-color: transparent; }
.sidebar {
  background: transparent; color: var(--text-default); display: flex; flex-direction: column; height: 100vh; position: sticky; top: 0;
}
.logo { height: 64px; display: flex; align-items: center; padding: 0 16px; border-bottom: 1px solid var(--border-color); }
.brand { font-weight: 600; font-size: 16px; letter-spacing: 0.3px; color: var(--text-strong); }
[data-theme="dark"] .brand { -webkit-text-stroke: 0.3px rgba(0,0,0,0.40); text-shadow: 0 1px 2px rgba(0,0,0,0.35); }
.menu { border-right: none; }
.topbar { height: 56px; display: flex; align-items: center; padding: 0 16px; background: transparent; border-bottom: 0; }
.spacer { flex: 1; }
.user-area .nav-user { margin-right: 8px; padding: 6px 10px; background: #f5f7fa; border-radius: var(--el-border-radius-round); font-size: 0.9rem; }
.content { max-width: 1280px; width: 100%; margin: 0 auto; padding: 10px 16px 16px; }
.app-footer { text-align: center; padding: 16px; background: var(--bg-footer); color: var(--text-default); font-size: 0.875rem; border-top: 1px solid var(--border-color); }
.breadcrumb-bar { padding: 6px 16px 0; }
/* settings drawer 样式已内置组件内 */

@media (max-width: 992px) { .sidebar { width: 200px !important; } }
@media (max-width: 768px) { .sidebar { display: none; } .content { padding: 12px; } }
</style>
