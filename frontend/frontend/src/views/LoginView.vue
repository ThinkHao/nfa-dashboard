<template>
  <div class="login-page">
    <el-card class="login-card" shadow="hover">
      <h2 class="page-title login-title">登录</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" autocomplete="current-password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="onSubmit" style="width: 100%">登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const formRef = ref()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function onSubmit() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    // 登录后加载用户信息（保险）
    await auth.loadProfile()
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.login-card {
  width: 380px;
  border-radius: 14px;
  overflow: hidden;
}
.login-title {
  text-align: center;
}

/* 浅色主题背景：简约大气的图片 + 渐变叠加增强可读性 */
:root:not([data-theme="dark"]) .login-page {
  background:
    radial-gradient(1200px 600px at 10% 10%, rgba(59,130,246,0.10), transparent 40%),
    radial-gradient(900px 500px at 90% 0%, rgba(14,165,233,0.08), transparent 40%),
    linear-gradient(135deg, rgba(248,250,252,0.90) 0%, rgba(238,242,247,0.90) 50%, rgba(248,250,252,0.90) 100%),
    url('https://images.unsplash.com/photo-1503264116251-35a269479413?ixlib=rb-4.0.3&auto=format&fit=crop&w=1920&q=80');
  background-size: cover;
  background-position: center;
  background-attachment: fixed;
}

/* 深色主题背景：更强覆盖让表单更聚焦 */
[data-theme="dark"] .login-page {
  background:
    radial-gradient(1200px 600px at 10% 10%, rgba(59,130,246,0.12), transparent 40%),
    radial-gradient(900px 500px at 90% 0%, rgba(14,165,233,0.10), transparent 40%),
    linear-gradient(135deg, rgba(11,18,32,0.86) 0%, rgba(14,27,46,0.86) 50%, rgba(11,18,32,0.86) 100%),
    url('https://images.unsplash.com/photo-1503264116251-35a269479413?ixlib=rb-4.0.3&auto=format&fit=crop&w=1920&q=80');
  background-size: cover;
  background-position: center;
  background-attachment: fixed;
}

/* 暗色下玻璃风格增强可读性 */
[data-theme="dark"] .login-card :deep(.el-card__body) {
  background: rgba(255,255,255,0.06) !important;
  -webkit-backdrop-filter: blur(10px);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255,255,255,0.12);
}

.login-card :deep(.el-form-item) { margin-bottom: 14px; }
</style>
