<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
const componentSize = computed(() => themeStore.size)
</script>

<template>
  <ElConfigProvider :size="componentSize" :locale="zhCn">
    <RouterView />
  </ElConfigProvider>
  
</template>

<style scoped>
</style>


