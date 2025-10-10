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
    <el-aside width="260px" class="sidebar glass-sidebar">
      <div class="logo glass-header">
        <div class="brand-mark">
          <span class="pulse"></span>
          <span class="mark-glow"></span>
        </div>
        <div class="brand-copy">
          <span class="brand-line brand-primary">NFA</span>
          <span class="brand-line brand-secondary">Dashboard</span>
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
        <p> 2025 学校流量监控系统 - NFA Dashboard</p>
      </el-footer>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  position: relative;
  background: transparent;
  overflow: hidden;
}
.layout::before,
.layout::after {
  content: '';
  position: absolute;
  inset: -20% -30% auto;
  height: 60%;
  background: radial-gradient(520px at 10% 25%, rgba(96, 165, 250, 0.18), transparent 65%);
  z-index: 0;
  pointer-events: none;
  transform: rotate(-4deg);
}
.layout::after {
  inset: auto -35% -25% 35%;
  height: 70%;
  background: radial-gradient(600px at 80% 80%, rgba(20, 184, 166, 0.22), transparent 70%);
  transform: rotate(6deg);
  filter: blur(2px);
}

.sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 16px 14px 24px;
  gap: 18px;
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.82) 0%, rgba(248, 250, 255, 0.65) 100%);
  border-right: 1px solid rgba(255, 255, 255, 0.45);
  backdrop-filter: saturate(160%) blur(18px);
  box-shadow: 0 12px 48px rgba(15, 23, 42, 0.08);
  z-index: 2;
}
:global([data-theme="dark"]) .sidebar {
  background: linear-gradient(180deg, rgba(12, 18, 31, 0.78) 0%, rgba(10, 19, 36, 0.62) 100%);
  border-right: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 18px 52px rgba(2, 6, 23, 0.45);
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 10px 12px 12px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  background: linear-gradient(120deg, rgba(255, 255, 255, 0.68) 0%, rgba(243, 247, 255, 0.32) 100%);
  position: relative;
  overflow: visible;
}
:global([data-theme="dark"]) .logo {
  background: linear-gradient(140deg, rgba(15, 23, 42, 0.85), rgba(36, 53, 99, 0.55));
  border: 1px solid rgba(59, 130, 246, 0.24);
}
.logo::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(120deg, rgba(59, 130, 246, 0.16), transparent 60%);
  opacity: 0.7;
}
.brand-mark {
  position: relative;
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  border-radius: 12px;
  border: 1px solid rgba(59, 130, 246, 0.35);
  background: radial-gradient(circle at 30% 30%, rgba(59, 130, 246, 0.8), rgba(37, 99, 235, 0.35));
  display: grid;
  place-items: center;
  overflow: hidden;
  box-shadow: inset 0 0 20px rgba(59, 130, 246, 0.3), 0 8px 18px rgba(59, 130, 246, 0.25);
}
.brand-mark .pulse {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #fff;
  position: relative;
  z-index: 2;
  box-shadow: 0 0 18px rgba(255, 255, 255, 0.75);
  animation: pulse 2.8s ease-in-out infinite;
}
.brand-mark .mark-glow {
  content: '';
  position: absolute;
  inset: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(96, 165, 250, 0.8), rgba(59, 130, 246, 0.25));
  filter: blur(12px);
  opacity: 0.9;
}
.brand-copy {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
  overflow: visible;
  flex: 1;
  gap: 4px;
}
.brand-line {
  font-weight: 700;
  line-height: 1;
  letter-spacing: 0.4px;
  color: var(--text-strong);
  text-transform: uppercase;
}
.brand-primary {
  font-size: 16px;
  opacity: 0.88;
}
.brand-secondary {
  font-size: 18px;
}

:deep(.el-menu) {
  border: none;
  background: transparent;
  padding: 4px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  border-radius: 12px !important;
  padding: 12px 16px !important;
  font-weight: 500;
  letter-spacing: 0.3px;
  transition: all 0.22s ease;
}
:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background: linear-gradient(90deg, rgba(148, 163, 184, 0.2), rgba(59, 130, 246, 0.12));
  transform: translateX(2px);
}
:deep(.el-menu-item.is-active:not(.is-disabled)) {
  background: linear-gradient(110deg, rgba(59, 130, 246, 0.22), rgba(59, 130, 246, 0.38));
  box-shadow: 0 12px 32px rgba(37, 99, 235, 0.24);
  color: #fff !important;
}

.breadcrumb-bar {
  position: relative;
  z-index: 1;
  padding: 12px 32px 0;
}

.content {
  position: relative;
  z-index: 1;
  max-width: 1360px;
  width: 100%;
  margin: 0 auto;
  padding: 24px 32px 32px;
}
.content::before {
  content: '';
  position: absolute;
  inset: 12px;
  border-radius: 28px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.42), rgba(248, 250, 255, 0.16));
  filter: blur(0.5px);
  z-index: -1;
  opacity: 0.75;
}
:global([data-theme="dark"]) .content::before {
  background: linear-gradient(140deg, rgba(12, 20, 35, 0.75), rgba(17, 37, 62, 0.45));
  border: 1px solid rgba(59, 130, 246, 0.22);
}

.app-footer {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: 20px 16px 32px;
  background: transparent;
  color: rgba(71, 85, 105, 0.8);
  font-size: 0.85rem;
  letter-spacing: 0.4px;
}
:global([data-theme="dark"]) .app-footer {
  color: rgba(203, 213, 225, 0.75);
}
.app-footer::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.08), rgba(20, 184, 166, 0.12), rgba(59, 130, 246, 0.08));
  opacity: 0.7;
  z-index: -1;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.4);
    opacity: 0.65;
  }
}

@media (max-width: 1200px) {
  .content::before { inset: 0; border-radius: 24px; }
}

@media (max-width: 992px) {
  .sidebar { width: 210px !important; padding: 16px; }
  .content { padding: 20px 24px 28px; }
}

@media (max-width: 768px) {
  .sidebar { display: none; }
  .content { padding: 16px 18px 24px; }
  .content::before { display: none; }
}
</style>
