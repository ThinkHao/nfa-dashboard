<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

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

function onLogout() {
  auth.logout()
}
</script>

<template>
  <ElConfigProvider>
    <el-container class="layout">
      <el-aside width="220px" class="sidebar glass-sidebar">
        <div class="logo">
          <span class="brand">NFA Dashboard</span>
        </div>
        <el-menu
          :default-active="$route.path"
          router
          background-color="transparent"
          text-color="#e5e7eb"
          active-text-color="#fff"
          class="menu"
        >
          <el-menu-item index="/">首页</el-menu-item>
          <el-menu-item v-if="canTraffic" index="/traffic">流量监控</el-menu-item>
          <el-menu-item v-if="canSchools" index="/schools">学校管理</el-menu-item>

          <el-sub-menu v-if="canSettlement" index="/settlement-group">
            <template #title>结算系统</template>
            <el-menu-item index="/settlement">结算面板</el-menu-item>
            <!-- 费率与业务对象页面预留，创建后放开（归属结算系统） -->
            <el-menu-item v-if="canRatesCustomer" index="/settlement/rates/customer">客户业务费率</el-menu-item>
            <el-menu-item v-if="canRatesNode" index="/settlement/rates/node">节点业务费率</el-menu-item>
            <el-menu-item v-if="canRatesFinal" index="/settlement/rates/final">最终客户费率</el-menu-item>
            <el-menu-item v-if="canEntities" index="/settlement/entities">业务对象</el-menu-item>
            <el-menu-item v-if="canBusinessTypes" index="/settlement/business-types">业务类型管理</el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="canOpLogs" index="/operation-logs">操作日志</el-menu-item>

          <el-sub-menu v-if="canSysUser || canSysRole || canSysPerm" index="/system-group">
            <template #title>系统管理</template>
            <el-menu-item v-if="canSysUser" index="/system/users">用户管理</el-menu-item>
            <el-menu-item v-if="canSysRole" index="/system/roles">角色管理</el-menu-item>
            <el-menu-item v-if="canSysPerm" index="/system/permissions">权限设置</el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="!isAuthed" index="/login">登录</el-menu-item>
        </el-menu>
      </el-aside>

      <el-container>
        <el-header class="topbar glass-header glass-surface">
          <div class="spacer"></div>
          <div class="user-area" v-if="isAuthed">
            <span class="nav-user">{{ auth.user?.alias || auth.user?.username }}</span>
            <el-button link type="primary" class="logout" @click="onLogout">退出</el-button>
          </div>
        </el-header>
        <el-main class="content">
          <RouterView />
        </el-main>
        <el-footer class="app-footer">
          <p>© 2025 学校流量监控系统 - NFA Dashboard</p>
        </el-footer>
      </el-container>
    </el-container>
  </ElConfigProvider>
  
</template>

<style scoped>
:root {
  --primary-color: #1890ff;
  --secondary-color: #52c41a;
  --dark-color: #001529;
  --light-color: #f0f2f5;
  --text-color: #333;
  --border-color: #e8e8e8;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.layout {
  min-height: 100vh;
  background-color: transparent;
}

.sidebar {
  background: transparent;
  color: #fff;
  display: flex;
  flex-direction: column;
  height: 100vh;
  position: sticky;
  top: 0;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.brand {
  font-weight: 600;
  font-size: 16px;
  letter-spacing: 0.3px;
}

.menu {
  border-right: none;
}

.topbar {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  background: transparent;
  border-bottom: 0;
}

.spacer {
  flex: 1;
}

.user-area .nav-user {
  margin-right: 8px;
  padding: 6px 10px;
  background: #f5f7fa;
  border-radius: var(--el-border-radius-round);
  font-size: 0.9rem;
}

.content {
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
  padding: 20px 24px 24px;
}

.app-footer {
  text-align: center;
  padding: 16px;
  background-color: #0b1220;
  color: white;
  font-size: 0.875rem;
}

@media (max-width: 992px) {
  .sidebar { width: 200px !important; }
}

@media (max-width: 768px) {
  .sidebar { display: none; }
  .content { padding: 12px; }
}
</style>
