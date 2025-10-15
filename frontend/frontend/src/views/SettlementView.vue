<template>
  <div class="settlement-view">
    <h1 class="page-title">结算中心</h1>
    <el-tabs v-model="activeTab" class="settlement-tabs">
      <el-tab-pane label="结算数据" name="data">
        <settlement-data-tab />
      </el-tab-pane>
      <el-tab-pane label="日95明细" name="dailyDetail">
        <settlement-daily-detail-tab />
      </el-tab-pane>
      <el-tab-pane label="结算任务" name="tasks">
        <settlement-tasks-tab />
      </el-tab-pane>
      <el-tab-pane label="结算配置" name="config">
        <settlement-config-tab />
      </el-tab-pane>
      <el-tab-pane label="结算公式" name="formula">
        <settlement-formula-tab />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SettlementDataTab from '../components/settlement/SettlementDataTab.vue'
import SettlementTasksTab from '../components/settlement/SettlementTasksTab.vue'
import SettlementConfigTab from '../components/settlement/SettlementConfigTab.vue'
import SettlementFormulaTab from '../components/settlement/SettlementFormulaTab.vue'
import SettlementDailyDetailTab from '../components/settlement/SettlementDailyDetailTab.vue'

// 当前激活的标签页
const activeTab = ref('data')
</script>

<style scoped>
.settlement-tabs {
  margin-bottom: 20px;
}
</style>
