<template>
  <div class="login-page">
    <div class="login-shell">
      <section class="login-hero">
        <div class="hero-signet">
          <span class="halo"></span>
          <span class="core">NFA</span>
        </div>
        <div class="hero-copy">
          <h1 class="hero-title">NFA Dashboard</h1>
          <p class="hero-subtitle">自研费率引擎与实时风控中枢，赋能大型教育集团的全链路流量与结算治理。</p>
          <ul class="hero-highlights">
            <li>
              <span class="dot"></span>
              <div class="text">双模式费率策略 · 自动与人工协同，分钟级生效</div>
            </li>
            <li>
              <span class="dot"></span>
              <div class="text">多维权限矩阵 · 精准控制用户、角色与操作日志</div>
            </li>
            <li>
              <span class="dot"></span>
              <div class="text">高保真可视化 · 结算、流量、院校画像一目了然</div>
            </li>
          </ul>
          <div class="hero-stats">
            <div class="stat-card">
              <span class="label">日请求峰值</span>
              <span class="value">8.6M</span>
            </div>
            <div class="stat-card">
              <span class="label">活跃院校</span>
              <span class="value">1,200+</span>
            </div>
            <div class="stat-card">
              <span class="label">费率策略</span>
              <span class="value">320+</span>
            </div>
          </div>
        </div>
      </section>

      <section class="login-panel glass-surface">
        <el-card class="login-card" shadow="never">
          <div class="card-head">
            <h2 class="page-title login-title">欢迎登录</h2>
            <p class="card-subtitle">请使用系统分配的账号完成身份验证</p>
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
  padding: clamp(24px, 5vw, 48px);
  background:
    radial-gradient(1200px 600px at 10% 10%, rgba(59, 130, 246, 0.18), transparent 45%),
    radial-gradient(1000px 600px at 90% 0%, rgba(20, 184, 166, 0.14), transparent 52%),
    linear-gradient(135deg, rgba(245, 248, 255, 0.92) 0%, rgba(237, 244, 252, 0.92) 50%, rgba(245, 248, 255, 0.92) 100%),
    url('https://images.unsplash.com/photo-1522199991221-88ee027bcefd?ixlib=rb-4.0.3&auto=format&fit=crop&w=1920&q=80');
  background-size: cover;
  background-position: center;
  background-attachment: fixed;
}

:global([data-theme="dark"]) .login-page {
  background:
    radial-gradient(1200px 600px at 10% 10%, rgba(59, 130, 246, 0.22), transparent 45%),
    radial-gradient(1000px 600px at 90% 0%, rgba(20, 184, 166, 0.18), transparent 52%),
    linear-gradient(135deg, rgba(10, 18, 32, 0.96) 0%, rgba(12, 26, 45, 0.96) 50%, rgba(10, 18, 32, 0.96) 100%),
    url('https://images.unsplash.com/photo-1522199991221-88ee027bcefd?ixlib=rb-4.0.3&auto=format&fit=crop&w=1920&q=80');
  background-size: cover;
  background-position: center;
  background-attachment: fixed;
}

.login-shell {
  max-width: 1120px;
  width: 100%;
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  gap: 48px;
  align-items: stretch;
  position: relative;
}

.login-hero {
  position: relative;
  padding: clamp(32px, 6vw, 52px);
  border-radius: 28px;
  background: linear-gradient(140deg, rgba(255, 255, 255, 0.82), rgba(224, 241, 255, 0.65));
  border: 1px solid rgba(59, 130, 246, 0.12);
  box-shadow: 0 25px 60px rgba(59, 130, 246, 0.18);
  overflow: hidden;
}

:global([data-theme="dark"]) .login-hero {
  background: linear-gradient(145deg, rgba(12, 22, 40, 0.88), rgba(26, 46, 78, 0.72));
  border: 1px solid rgba(59, 130, 246, 0.24);
  box-shadow: 0 30px 80px rgba(2, 6, 23, 0.55);
}

.login-hero::after {
  content: '';
  position: absolute;
  inset: -20% -15% auto auto;
  width: 320px;
  height: 320px;
  background: radial-gradient(circle, rgba(59, 130, 246, 0.22) 0%, rgba(59, 130, 246, 0) 70%);
  filter: blur(6px);
  transform: rotate(18deg);
}

.hero-signet {
  position: relative;
  width: 84px;
  height: 84px;
  border-radius: 26px;
  background: linear-gradient(135deg, rgba(30, 64, 175, 0.85), rgba(37, 99, 235, 0.55));
  display: grid;
  place-items: center;
  margin-bottom: 28px;
  box-shadow: inset 0 0 28px rgba(59, 130, 246, 0.38), 0 16px 40px rgba(59, 130, 246, 0.4);
  overflow: hidden;
}

.hero-signet .halo {
  position: absolute;
  inset: -30%;
  background: radial-gradient(circle, rgba(96, 165, 250, 0.55), transparent 65%);
  animation: breathe 6s ease-in-out infinite;
}

.hero-signet .core {
  position: relative;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 4px;
  color: #fff;
  z-index: 1;
}

.hero-copy {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.hero-title {
  font-size: clamp(30px, 3.2vw, 42px);
  font-weight: 800;
  letter-spacing: 0.6px;
  color: var(--text-strong);
  margin: 0;
}

:global([data-theme="dark"]) .hero-title {
  color: #e2e8f0;
  text-shadow: 0 14px 30px rgba(15, 23, 42, 0.55);
}

.hero-subtitle {
  font-size: 16px;
  line-height: 1.7;
  color: rgba(30, 64, 175, 0.75);
  margin: 0;
  max-width: 420px;
}

:global([data-theme="dark"]) .hero-subtitle {
  color: rgba(191, 219, 254, 0.78);
}

.hero-highlights {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.hero-highlights li {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  color: rgba(30, 41, 59, 0.82);
}

:global([data-theme="dark"]) .hero-highlights li {
  color: rgba(203, 213, 225, 0.82);
}

.hero-highlights .dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(59, 130, 246, 1), rgba(14, 165, 233, 0.8));
  margin-top: 6px;
  box-shadow: 0 0 14px rgba(59, 130, 246, 0.45);
}

.hero-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  margin-top: 12px;
}

.stat-card {
  padding: 18px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(148, 163, 184, 0.2);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.45);
  text-align: left;
}

:global([data-theme="dark"]) .stat-card {
  background: rgba(30, 41, 59, 0.45);
  border: 1px solid rgba(63, 131, 248, 0.32);
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.45);
}

.stat-card .label {
  font-size: 12px;
  letter-spacing: 1.8px;
  text-transform: uppercase;
  color: rgba(71, 85, 105, 0.75);
}

:global([data-theme="dark"]) .stat-card .label {
  color: rgba(148, 163, 184, 0.72);
}

.stat-card .value {
  display: block;
  margin-top: 8px;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.4px;
  color: var(--text-strong);
}

:global([data-theme="dark"]) .stat-card .value {
  color: rgba(226, 232, 240, 0.95);
}

.login-panel {
  position: relative;
  border-radius: 30px;
  padding: 3px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.35), rgba(14, 165, 233, 0.22));
  box-shadow: 0 25px 70px rgba(15, 23, 42, 0.18);
}

.login-card {
  border-radius: 26px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(148, 163, 184, 0.28);
}

:global([data-theme="dark"]) .login-card {
  background: rgba(12, 20, 35, 0.92);
  border: 1px solid rgba(59, 130, 246, 0.28);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);
}

.login-card :deep(.el-card__body) {
  padding: clamp(32px, 5vw, 48px);
}

.card-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
  text-align: left;
}

.login-title {
  margin: 0;
  text-align: left;
}

.card-subtitle {
  margin: 0;
  color: rgba(71, 85, 105, 0.82);
  font-size: 14px;
}

:global([data-theme="dark"]) .card-subtitle {
  color: rgba(203, 213, 225, 0.78);
}

.login-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.login-form :deep(.el-form-item__label) {
  font-weight: 600;
  letter-spacing: 0.4px;
  color: rgba(71, 85, 105, 0.9) !important;
}

:global([data-theme="dark"]) .login-form :deep(.el-form-item__label) {
  color: rgba(226, 232, 240, 0.92) !important;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 14px;
  padding: 4px 14px;
  box-shadow: 0 1px 0 rgba(148, 163, 184, 0.18);
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.18);
}

.form-footer {
  margin-top: 14px;
}

.submit-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  letter-spacing: 0.6px;
  border-radius: 14px;
  background-image: linear-gradient(120deg, #2563eb, #3b82f6);
  box-shadow: 0 18px 40px rgba(37, 99, 235, 0.25);
}

.submit-btn:hover {
  filter: brightness(1.05);
  box-shadow: 0 22px 50px rgba(37, 99, 235, 0.35);
}

@keyframes breathe {
  0%, 100% {
    transform: scale(1);
    opacity: 0.65;
  }
  50% {
    transform: scale(1.18);
    opacity: 1;
  }
}

@media (max-width: 1080px) {
  .login-shell {
    grid-template-columns: 1fr;
    gap: 32px;
  }
  .login-hero {
    order: 2;
  }
  .login-panel {
    order: 1;
  }
}

@media (max-width: 640px) {
  .login-page {
    padding: 24px 16px;
    background-attachment: scroll;
  }
  .login-hero {
    padding: 26px;
    border-radius: 20px;
  }
  .hero-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .login-card {
    border-radius: 20px;
  }
  .login-card :deep(.el-card__body) {
    padding: 28px;
  }
}
</style>
