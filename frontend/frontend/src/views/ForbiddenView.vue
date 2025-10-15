<template>
  <div class="forbidden">
    <el-result icon="warning" title="无权限访问" sub-title="您没有访问该页面的权限，请联系管理员分配权限或切换账号。">
      <template #extra>
        <el-button type="primary" @click="goHome">返回首页</el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
function goHome() {
  router.push('/')
}
</script>

<style scoped>
.forbidden { padding: 40px; display: flex; justify-content: center; }
</style>
