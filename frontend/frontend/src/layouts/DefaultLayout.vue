<script setup lang="ts">
import { RouterView } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import Breadcrumb from '@/components/Breadcrumb/index.vue'
import TagsView from '@/components/TagsView/index.vue'
import SidebarMenu from '@/components/Sidebar/Menu.vue'
import Navbar from '@/components/Navbar/index.vue'

const currentYear = new Date().getFullYear()

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
    <el-aside width="240px" class="sidebar">
      <div class="logo">
        <div class="brand-mark">N</div>
        <div class="brand-copy">
          <span class="brand-line brand-primary">NFA Dashboard</span>
          <span class="brand-line brand-secondary">运营工作台</span>
        </div>
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
        <div class="footer-inner">
          <span>© {{ currentYear }} NFA Dashboard · 学校流量监控系统</span>
        </div>
      </el-footer>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  background: var(--bg-page);
}

.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  height: 100vh;
  width: 240px;
  display: flex;
  flex-direction: column;
  padding: 16px 12px;
  gap: 14px;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-color);
  z-index: 30;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  background: var(--bg-card);
}
.brand-mark {
  width: 34px;
  height: 34px;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 15px;
  font-weight: 700;
  flex-shrink: 0;
}
.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.brand-line {
  font-weight: 600;
  line-height: 1;
  color: var(--text-default);
}
.brand-primary {
  font-size: 14px;
}
.brand-secondary {
  color: var(--text-muted);
  font-size: 12px;
}

:deep(.el-menu) {
  padding: 2px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  border-radius: var(--radius-sm) !important;
  padding: 10px 14px !important;
  font-weight: 500;
  letter-spacing: 0.01em;
}

.breadcrumb-bar {
  padding: 10px 28px 0;
  margin-left: 240px;
}

.content {
  width: calc(100vw - 240px);
  max-width: none;
  margin: 0 0 0 240px;
  padding: 16px 28px 24px;
}

.app-footer {
  margin-left: 240px;
  text-align: center;
  padding: 12px 16px 18px;
  font-size: 12px;
}

.footer-inner {
  max-width: none;
  margin: 0 auto;
}

@media (max-width: 992px) {
  .sidebar {
    width: 216px;
  }
  .breadcrumb-bar {
    margin-left: 216px;
    padding: 10px 20px 0;
  }
  .content {
    width: calc(100vw - 216px);
    margin-left: 216px;
    padding: 14px 20px 20px;
  }
  .app-footer {
    margin-left: 216px;
  }
}

@media (max-width: 768px) {
  .sidebar { display: none; }
  .breadcrumb-bar {
    margin-left: 0;
    padding: 8px 14px 0;
  }
  .content {
    width: 100%;
    margin-left: 0;
    padding: 12px 14px 18px;
  }
  .app-footer {
    margin-left: 0;
  }
}
</style>


