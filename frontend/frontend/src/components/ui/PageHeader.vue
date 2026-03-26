<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <header class="page-header">
    <h1 class="page-title">{{ title }}</h1>
    <p v-if="description" class="page-desc">{{ description }}</p>
  </header>
</template>

<style scoped>
.page-header {
  margin-bottom: var(--space-4);
}

.page-desc {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}
</style>


