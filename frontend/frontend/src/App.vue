<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'
import BackgroundTasks from '@/components/BackgroundTasks.vue'

const themeStore = useThemeStore()
const componentSize = computed(() => themeStore.size)
</script>

<template>
  <ElConfigProvider :size="componentSize">
    <BackgroundTasks />
    <RouterView />
  </ElConfigProvider>
  
</template>

<style scoped>
</style>
