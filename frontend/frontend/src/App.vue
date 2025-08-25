<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
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

function onLogout() {
  auth.logout()
}
</script>

<template>
  <ElConfigProvider>
    <div class="app-container">
      <header class="app-header app-header--solid">
        <div class="logo-container">
          <h1 class="app-title">学校流量监控系统</h1>
        </div>
        <nav class="main-nav">
          <RouterLink to="/" class="nav-link">首页</RouterLink>
          <RouterLink to="/traffic" class="nav-link" v-if="canTraffic">流量监控</RouterLink>
          <RouterLink to="/schools" class="nav-link" v-if="canSchools">学校管理</RouterLink>
          <RouterLink to="/settlement" class="nav-link" v-if="canSettlement">结算系统</RouterLink>
          <RouterLink to="/operation-logs" class="nav-link" v-if="canOpLogs">操作日志</RouterLink>
          <RouterLink to="/system/users" class="nav-link" v-if="canSysUser">用户管理</RouterLink>
          <RouterLink to="/system/roles" class="nav-link" v-if="canSysRole">角色管理</RouterLink>
          <RouterLink to="/system/permissions" class="nav-link" v-if="canSysPerm">权限设置</RouterLink>
          <RouterLink to="/login" class="nav-link" v-if="!isAuthed">登录</RouterLink>
          <span class="nav-user" v-if="isAuthed">{{ auth.user?.alias || auth.user?.username }}</span>
          <a href="javascript:void(0)" class="nav-link logout" v-if="isAuthed" @click="onLogout">退出</a>
        </nav>
      </header>
      
      <main class="app-main page-container">
        <RouterView />
      </main>
      
      <footer class="app-footer">
        <p>© 2025 学校流量监控系统 - NFA Dashboard</p>
      </footer>
    </div>
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

.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: var(--light-color);
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 2rem;
  height: 64px;
  background-color: transparent; /* default, overridden by modifiers */
  color: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.app-header--solid {
  background-color: var(--dark-color);
  border-bottom: 1px solid rgba(255,255,255,0.06);
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
}

.app-title {
  font-size: 1.5rem;
  font-weight: 600;
}

.main-nav {
  display: flex;
  gap: 1.5rem;
}

.nav-link {
  color: #e5e7eb; /* gray-200 for contrast on dark */
  text-decoration: none;
  font-size: 0.95rem;
  font-weight: 500;
  padding: 8px 14px;
  border-radius: var(--el-border-radius-round);
  border: 1px solid transparent;
  transition: all 0.2s ease;
}

.nav-link:hover { background-color: rgba(255, 255, 255, 0.14); }
.nav-link:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 2px; }

.router-link-active { background-color: var(--el-color-primary); color: #fff; box-shadow: var(--el-box-shadow-light); }

.app-main {
  flex: 1;
  padding: 2rem 0;
}

/* Ensure content below header is centered and constrained */
.app-main.page-container {
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
  padding-left: 24px;
  padding-right: 24px;
}

.app-footer {
  text-align: center;
  padding: 1.5rem;
  background-color: #0b1220;
  color: white;
  font-size: 0.875rem;
}

.nav-user {
  margin-left: 4px;
  padding: 6px 10px;
  background: rgba(255,255,255,0.12);
  border-radius: var(--el-border-radius-round);
  font-size: 0.9rem;
}

@media (max-width: 768px) {
  .app-header {
    flex-direction: column;
    height: auto;
    padding: 1rem;
  }
  
  .main-nav {
    margin-top: 1rem;
    width: 100%;
    justify-content: center;
  }
  
  .app-main {
    padding: 1rem;
  }
}
</style>
