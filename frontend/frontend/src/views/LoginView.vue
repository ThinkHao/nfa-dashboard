<template>
  <div class="login-page">
    <div class="login-shell">
      <section class="login-brand">
        <div class="brand-mark">NFA</div>
        <h1 class="brand-title">NFA Dashboard</h1>
        <p class="brand-subtitle">学校流量监控与结算管理后台</p>
      </section>

      <section class="login-panel">
        <el-card class="login-card" shadow="never">
          <div class="card-head">
            <h2 class="page-title login-title">登录系统</h2>
            <p class="card-subtitle">请输入账号信息，登录后进入工作台</p>
          </div>
          <el-form
            :model="form"
            :rules="rules"
            ref="formRef"
            label-position="top"
            size="large"
            class="login-form"
          >
            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" placeholder="请输入用户名" autocomplete="username">
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="请输入密码"
                autocomplete="current-password"
                show-password
              >
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <div class="form-footer">
              <el-button type="primary" :loading="loading" @click="onSubmit" class="submit-btn">进入控制台</el-button>
            </div>
          </el-form>
        </el-card>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { User, Lock } from '@element-plus/icons-vue'

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
  padding: clamp(20px, 4vw, 36px);
  background: linear-gradient(180deg, var(--bg-page) 0%, color-mix(in srgb, var(--bg-page) 92%, white) 100%);
}

.login-shell {
  max-width: 920px;
  width: 100%;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  align-items: stretch;
}

.login-brand {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  padding: clamp(28px, 5vw, 42px);
  box-shadow: var(--shadow-1);
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 14px;
}

.brand-mark {
  width: 54px;
  height: 54px;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.03em;
}

.brand-title {
  margin: 0;
  color: var(--text-strong);
  font-size: clamp(28px, 3vw, 34px);
  font-weight: 700;
}

.brand-subtitle {
  margin: 0;
  color: var(--text-muted);
  line-height: 1.6;
}

.login-panel {
  border-radius: var(--radius-lg);
}

.login-card {
  height: 100%;
}

.login-card :deep(.el-card__body) {
  height: 100%;
  padding: clamp(28px, 4vw, 36px);
}

.card-head {
  margin-bottom: 12px;
}

.login-title {
  margin: 0 0 6px;
}

.card-subtitle {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.form-footer {
  margin-top: 8px;
}

.submit-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
}

@media (max-width: 900px) {
  .login-shell {
    grid-template-columns: 1fr;
  }

  .login-brand {
    padding: 24px;
  }
}
</style>


