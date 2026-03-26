<script setup lang="ts">
defineProps<{
  title?: string
}>()
</script>

<template>
  <el-card class="section-card" shadow="never">
    <template v-if="title" #header>
      <div class="section-head">
        <span class="section-title">{{ title }}</span>
      </div>
    </template>
    <slot />
  </el-card>
</template>

<style scoped>
.section-card {
  margin-bottom: var(--space-4);
}

.section-title {
  font-weight: 600;
  color: var(--text-default);
}
</style>


