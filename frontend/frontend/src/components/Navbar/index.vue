<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { Moon, Sunny, SwitchButton, Brush, Lock } from '@element-plus/icons-vue'
import SettingsDrawer from '@/components/Settings/SettingsDrawer.vue'
import BackgroundTasks from '@/components/BackgroundTasks.vue'

const auth = useAuthStore()
function onLogout() { auth.logout() }

const theme = useThemeStore()
const isDark = computed(() => theme.isDark)
function toggleDark() { theme.toggleDark() }

const pwdDialogVisible = ref(false)
const pwdLoading = ref(false)
const pwdFormRef = ref()
const pwdForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

const pwdRules = {
  old_password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    {
      validator: (_: unknown, value: string, callback: (error?: Error) => void) => {
        if (!value || value.length < 8) return callback(new Error('新密码至少 8 位'))
        if (!/[a-z]/.test(value) || !/[A-Z]/.test(value) || !/\d/.test(value) || !/[^A-Za-z0-9]/.test(value)) {
          return callback(new Error('需包含大小写字母、数字和符号'))
        }
        callback()
      }, trigger: 'blur'
    },
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (_: unknown, value: string, callback: (error?: Error) => void) => {
        if (value !== pwdForm.new_password) return callback(new Error('两次输入的新密码不一致'))
        callback()
      }, trigger: 'blur'
    },
  ],
}

function resetPwdForm() {
  pwdForm.old_password = ''
  pwdForm.new_password = ''
  pwdForm.confirm_password = ''
  pwdFormRef.value?.clearValidate?.()
}

function openPwdDialog() {
  resetPwdForm()
  pwdDialogVisible.value = true
}

async function submitChangePassword() {
  await pwdFormRef.value?.validate()
  pwdLoading.value = true
  try {
    await api.auth.changePassword({
      old_password: pwdForm.old_password,
      new_password: pwdForm.new_password,
    })
    ElMessage.success('密码修改成功，请重新登录')
    pwdDialogVisible.value = false
    auth.logout()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '修改密码失败')
  } finally {
    pwdLoading.value = false
  }
}
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
        <el-tooltip effect="dark" content="修改密码" placement="bottom">
          <el-button link circle class="icon-btn" @click="openPwdDialog">
            <el-icon><Lock /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip effect="dark" content="退出登录" placement="bottom">
          <el-button link circle class="icon-btn logout" @click="onLogout">
            <el-icon><SwitchButton /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>
  </el-header>

  <el-dialog
    v-model="pwdDialogVisible"
    title="修改密码"
    width="460px"
    :close-on-click-modal="false"
    @closed="resetPwdForm"
  >
    <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-position="top">
      <el-form-item label="当前密码" prop="old_password">
        <el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" />
      </el-form-item>
      <el-form-item label="确认新密码" prop="confirm_password">
        <el-input v-model="pwdForm.confirm_password" type="password" show-password autocomplete="new-password" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="pwdDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="pwdLoading" @click="submitChangePassword">确认修改</el-button>
    </template>
  </el-dialog>
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

