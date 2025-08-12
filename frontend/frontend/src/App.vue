<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const isAuthed = computed(() => auth.isAuthenticated)
const canOpLogs = computed(() => auth.hasPermission('operation_logs.read'))
const canSettlement = computed(() => auth.hasPermission('settlement.calculate'))
const canSysUser = computed(() => auth.hasPermission('system.user.manage'))
const canSysRole = computed(() => auth.hasPermission('system.role.manage'))

function onLogout() {
  auth.logout()
}
</script>

<template>
  <ElConfigProvider>
    <div class="app-container">
      <header class="app-header">
        <div class="logo-container">
          <h1 class="app-title">学校流量监控系统</h1>
        </div>
        <nav class="main-nav">
          <RouterLink to="/" class="nav-link">首页</RouterLink>
          <RouterLink to="/traffic" class="nav-link">流量监控</RouterLink>
          <RouterLink to="/schools" class="nav-link">学校管理</RouterLink>
          <RouterLink to="/settlement" class="nav-link" v-if="canSettlement">结算系统</RouterLink>
          <RouterLink to="/operation-logs" class="nav-link" v-if="canOpLogs">操作日志</RouterLink>
          <RouterLink to="/system/users" class="nav-link" v-if="canSysUser">用户管理</RouterLink>
          <RouterLink to="/system/roles" class="nav-link" v-if="canSysRole">角色管理</RouterLink>
          <RouterLink to="/system/permissions" class="nav-link" v-if="canSysRole">权限设置</RouterLink>
          <RouterLink to="/login" class="nav-link" v-if="!isAuthed">登录</RouterLink>
          <span class="nav-user" v-if="isAuthed">{{ auth.user?.display_name || auth.user?.username }}</span>
          <a href="javascript:void(0)" class="nav-link logout" v-if="isAuthed" @click="onLogout">退出</a>
        </nav>
      </header>
      
      <main class="app-main">
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
  background-color: var(--dark-color);
  color: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
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
  color: white;
  text-decoration: none;
  font-size: 1rem;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.nav-link:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.router-link-active {
  background-color: var(--primary-color);
  color: white;
}

.app-main {
  flex: 1;
  padding: 2rem;
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
}

.app-footer {
  text-align: center;
  padding: 1.5rem;
  background-color: var(--dark-color);
  color: white;
  font-size: 0.875rem;
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
