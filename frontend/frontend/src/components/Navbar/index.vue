<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { computed } from 'vue'
import { Moon, Sunny, SwitchButton, Brush } from '@element-plus/icons-vue'
import SettingsDrawer from '@/components/Settings/SettingsDrawer.vue'

const auth = useAuthStore()
function onLogout() { auth.logout() }

const theme = useThemeStore()
const isDark = computed(() => theme.isDark)
function toggleDark() { theme.toggleDark() }
</script>

<template>
  <el-header class="topbar glass-header glass-surface">
    <div class="left">
      <!-- 预留左侧：系统名称/折叠按钮/搜索等 -->
    </div>
    <div class="spacer"></div>
    <div class="right">
      <div class="user-area" v-if="auth.isAuthenticated">
        <span class="nav-user">{{ auth.user?.alias || auth.user?.username }}</span>
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
.topbar { height: 56px; display: flex; align-items: center; padding: 0 16px; background: transparent; border-bottom: 0; }
.spacer { flex: 1; }
.user-area { display: flex; align-items: center; gap: 6px; }
.user-area .nav-user { margin-right: 6px; padding: 6px 10px; border-radius: var(--el-border-radius-round); font-size: 0.9rem; font-weight: 600; }
:root:not([data-theme="dark"]) .user-area .nav-user { background: #f5f7fa; color: var(--text-color); border: 1px solid var(--border-color); }
[data-theme="dark"] .user-area .nav-user { background: rgba(255,255,255,0.12); color: var(--text-default); border: 1px solid rgba(255,255,255,0.18); }
.icon-btn :deep(.el-icon) { font-size: 18px; }
</style>
