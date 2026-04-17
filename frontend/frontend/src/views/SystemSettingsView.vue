<template>
  <div class="page-container">
    <PageHeader title="系统设置" description="管理全局业务开关与数据口径策略。" />

    <SectionCard title="流量监控数据口径">
      <div class="setting-row" v-loading="loading">
        <div>
          <div class="setting-title">隐藏不参与结算院校</div>
          <div class="setting-desc">
            开启后，流量监控仅显示同时满足“流量可见范围”和“参与结算规则”的院校数据。
          </div>
        </div>
        <el-switch v-model="form.hide_non_settlement_schools_in_traffic" />
      </div>

      <div class="actions">
        <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
        <el-button :disabled="loading || saving" @click="load">重置</el-button>
      </div>
    </SectionCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import type { SystemTrafficSettings } from '@/types/api'

const loading = ref(false)
const saving = ref(false)
const form = reactive<SystemTrafficSettings>({
  hide_non_settlement_schools_in_traffic: false,
})

async function load() {
  loading.value = true
  try {
    const cfg = await api.system.settings.getTraffic()
    form.hide_non_settlement_schools_in_traffic = !!cfg?.hide_non_settlement_schools_in_traffic
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载系统设置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload: SystemTrafficSettings = {
      hide_non_settlement_schools_in_traffic: !!form.hide_non_settlement_schools_in_traffic,
    }
    const cfg = await api.system.settings.updateTraffic(payload)
    form.hide_non_settlement_schools_in_traffic = !!cfg?.hide_non_settlement_schools_in_traffic
    ElMessage.success('保存成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存系统设置失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.setting-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-default);
}

.setting-desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--text-muted);
  max-width: 680px;
}

.actions {
  margin-top: 16px;
  display: flex;
  gap: 10px;
}
</style>
