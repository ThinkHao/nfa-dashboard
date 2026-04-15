<template>
  <el-button type="primary" @click="$emit('trigger')">
    <el-icon v-if="running" class="is-loading mr-4"><Loading /></el-icon>
    <span>{{ running ? '取消查询' : '查询' }}</span>
  </el-button>
</template>

<script setup lang="ts">
import { Loading } from '@element-plus/icons-vue'

defineProps<{ running: boolean }>()
defineEmits<{ (e: 'trigger'): void }>()
</script>

<style scoped>
.mr-4 {
  margin-right: 4px;
}
</style>
