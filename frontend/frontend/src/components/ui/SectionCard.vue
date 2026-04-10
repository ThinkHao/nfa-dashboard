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
        <div v-if="$slots.actions" class="section-actions">
          <slot name="actions" />
        </div>
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

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
</style>


