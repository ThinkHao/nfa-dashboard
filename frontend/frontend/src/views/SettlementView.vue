<template>
  <div class="settlement-view">
    <el-tabs v-model="activeTab" class="settlement-tabs">
      <el-tab-pane label="结算数据" name="data">
        <settlement-data-tab />
      </el-tab-pane>
      <el-tab-pane label="结算任务" name="tasks">
        <settlement-tasks-tab />
      </el-tab-pane>
      <el-tab-pane label="结算配置" name="config">
        <settlement-config-tab />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SettlementDataTab from '../components/settlement/SettlementDataTab.vue'
import SettlementTasksTab from '../components/settlement/SettlementTasksTab.vue'
import SettlementConfigTab from '../components/settlement/SettlementConfigTab.vue'

// 当前激活的标签页
const activeTab = ref('data')
</script>

<style scoped>
.settlement-view {
  padding: 20px;
}

.settlement-tabs {
  margin-bottom: 20px;
}
</style>
