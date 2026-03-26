<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { computed } from 'vue'
import { Moon, Sunny, SwitchButton, Brush } from '@element-plus/icons-vue'
import SettingsDrawer from '@/components/Settings/SettingsDrawer.vue'
import BackgroundTasks from '@/components/BackgroundTasks.vue'

const auth = useAuthStore()
function onLogout() { auth.logout() }

const theme = useThemeStore()
const isDark = computed(() => theme.isDark)
function toggleDark() { theme.toggleDark() }
</script>

<template>
  <el-header class="topbar">
    <div class="left">
      <span class="workspace-label">控制台</span>
    </div>
    <div class="spacer"></div>
    <div class="right">
      <div class="user-area" v-if="auth.isAuthenticated">
        <span class="nav-user">{{ auth.user?.alias || auth.user?.username }}</span>
        <BackgroundTasks inline />
        <el-tooltip effect="dark" content="切换主题" placement="bottom">
          <el-button link circle class="icon-btn" @click="toggleDark">
            <el-icon><component :is="isDark ? Sunny : Moon" /></el-icon>
          </el-button>
        </el-tooltip>
        <SettingsDrawer>
          <template #reference>
            <el-button link circle class="icon-btn" title="外观">
              <el-icon><Brush /></el-icon>
            </el-button>
          </template>
        </SettingsDrawer>
        <el-tooltip effect="dark" content="退出登录" placement="bottom">
          <el-button link circle class="icon-btn logout" @click="onLogout">
            <el-icon><SwitchButton /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>
  </el-header>
</template>

<style scoped>
.topbar {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  margin-left: 240px;
  background: var(--bg-header);
  border-bottom: 1px solid var(--border-color);
}
.spacer { flex: 1; }
.user-area { display: flex; align-items: center; gap: 6px; }
.workspace-label {
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.user-area .nav-user {
  margin-right: 6px;
  padding: 6px 10px;
  border-radius: var(--el-border-radius-round);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-default);
  background: var(--bg-subtle);
  border: 1px solid var(--border-color);
}
.icon-btn :deep(.el-icon) { font-size: 18px; }

@media (max-width: 992px) {
  .topbar {
    margin-left: 216px;
    padding: 0 14px;
  }
}

@media (max-width: 768px) {
  .topbar {
    margin-left: 0;
  }
}
</style>


