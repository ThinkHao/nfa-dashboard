<template>
  <div class="settlement-dashboard">
    <h1 class="page-title">结算系统</h1>
    <el-card class="dashboard-card" shadow="never">
      <settlement-results-tab />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import SettlementResultsTab from '@/components/settlement/SettlementResultsTab.vue'
</script>

<style scoped>
.dashboard-card {
  margin-top: 16px;
}

.dashboard-tabs {
  margin-top: 8px;
}
</style>
