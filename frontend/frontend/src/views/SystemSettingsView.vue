<template>
  <div class="page-container">
    <PageHeader title="系统设置" description="管理全局业务开关与数据口径策略。" />

    <SectionCard title="流速/流量监控数据口径">
      <div class="setting-row" v-loading="loading">
        <div>
          <div class="setting-title">隐藏不参与结算院校</div>
          <div class="setting-desc">
            开启后，流速和流量监控仅显示同时满足“流量可见范围”和“参与结算规则”的院校数据。
          </div>
        </div>
        <el-switch v-model="form.hide_non_settlement_schools_in_traffic" />
      </div>

      <div class="setting-row">
        <div>
          <div class="setting-title">流速监控字节进制</div>
          <div class="setting-desc">用于流速监控页面的字节单位换算（B/KB/MB/GB）。</div>
        </div>
        <el-select v-model="form.traffic_byte_unit_base" class="field-w-180">
          <el-option :value="1000" label="1000（SI）" />
          <el-option :value="1024" label="1024（IEC）" />
        </el-select>
      </div>

      <div class="setting-row">
        <div>
          <div class="setting-title">日流量监控字节进制</div>
          <div class="setting-desc">用于流量监控页面的日流量换算；选择1000可与 Grafana bytes(SI) 保持一致。</div>
        </div>
        <el-select v-model="form.daily_traffic_volume_unit_base" class="field-w-180">
          <el-option :value="1000" label="1000（SI，与 Grafana 一致）" />
          <el-option :value="1024" label="1024（IEC）" />
        </el-select>
      </div>

      <div class="setting-row">
        <div>
          <div class="setting-title">院校结算结果 G 单位进制</div>
          <div class="setting-desc">仅作用于院校（NFA）结算结果：院校结算查询的日95流量值与金额按此 GB(1000)/GiB(1024) 口径换算，二者同口径便于对账。EDC 节点结算使用各自的进制设置，不受此项影响。</div>
        </div>
        <el-select v-model="form.settlement_result_unit_base" class="field-w-180">
          <el-option :value="1000" label="1000（GB）" />
          <el-option :value="1024" label="1024（GiB）" />
        </el-select>
      </div>

      <div class="setting-row">
        <div>
          <div class="setting-title">结算数据页 95 值单位</div>
          <div class="setting-desc">影响结算中心“结算数据”页的 95 值显示与导出。</div>
        </div>
        <el-select v-model="form.settlement_data_rate_unit" class="field-w-180">
          <el-option value="Mbps" label="Mbps" />
          <el-option value="Gbps" label="Gbps" />
        </el-select>
      </div>


      <div class="setting-row">
        <div>
          <div class="setting-title">院校结算查询页 95 值单位</div>
          <div class="setting-desc">影响“院校结算查询”页的 95 值显示与导出。</div>
        </div>
        <el-select v-model="form.settlement_single_user_rate_unit" class="field-w-180">
          <el-option value="Mbps" label="Mbps" />
          <el-option value="Gbps" label="Gbps" />
        </el-select>
      </div>

      <div class="actions">
        <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
        <el-button :disabled="loading || saving" @click="load">重置</el-button>
      </div>
    </SectionCard>

    <SectionCard title="客户费率待办提醒">
      <div class="setting-row" v-loading="todoLoading">
        <div>
          <div class="setting-title">待办接收人</div>
          <div class="setting-desc">
            新院校产生客户费率记录后，若客户费、线路费或节点通用费任一单价为空，会持续提醒所选用户。接收人需要客户业务费率查看和维护权限；数值 0 视为已填写。
          </div>
        </div>
        <el-select
          v-model="todoUserIDs"
          multiple
          filterable
          clearable
          collapse-tags
          collapse-tags-tooltip
          placeholder="搜索并选择一个或多个用户"
          class="todo-user-select"
        >
          <el-option v-for="user in todoUserOptions" :key="user.id" :label="user.label" :value="user.id" />
        </el-select>
      </div>
      <div class="actions">
        <el-button type="primary" :loading="todoSaving" @click="saveTodoSettings">保存提醒设置</el-button>
        <el-button :disabled="todoLoading || todoSaving" @click="loadTodoSettings">重置</el-button>
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
import type { SystemTrafficSettings, SystemUser } from '@/types/api'
import { useSystemTrafficSettings } from '@/composables/useSystemTrafficSettings'

const loading = ref(false)
const saving = ref(false)
const todoLoading = ref(false)
const todoSaving = ref(false)
const todoUserIDs = ref<number[]>([])
const todoUserOptions = ref<{ id: number; label: string }[]>([])
const trafficSettings = useSystemTrafficSettings()
const form = reactive<SystemTrafficSettings>({
  hide_non_settlement_schools_in_traffic: false,
  traffic_byte_unit_base: 1024,
  daily_traffic_volume_unit_base: 1000,
  settlement_result_unit_base: 1024,
  settlement_data_rate_unit: 'Mbps',
  settlement_daily_detail_rate_unit: 'Mbps',
  settlement_single_user_rate_unit: 'Gbps',
})

async function load() {
  loading.value = true
  try {
    const cfg = await api.system.settings.getTraffic()
    form.hide_non_settlement_schools_in_traffic = !!cfg?.hide_non_settlement_schools_in_traffic
    form.traffic_byte_unit_base = cfg?.traffic_byte_unit_base === 1000 ? 1000 : 1024
    form.daily_traffic_volume_unit_base = cfg?.daily_traffic_volume_unit_base === 1024 ? 1024 : 1000
    form.settlement_result_unit_base = cfg?.settlement_result_unit_base === 1000 ? 1000 : 1024
    form.settlement_data_rate_unit = cfg?.settlement_data_rate_unit === 'Gbps' ? 'Gbps' : 'Mbps'
    form.settlement_daily_detail_rate_unit = cfg?.settlement_daily_detail_rate_unit === 'Gbps' ? 'Gbps' : 'Mbps'
    form.settlement_single_user_rate_unit = cfg?.settlement_single_user_rate_unit === 'Mbps' ? 'Mbps' : 'Gbps'
    trafficSettings.apply(cfg)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载系统设置失败')
  } finally {
    loading.value = false
  }
  await loadTodoSettings()
}

function userLabel(user: SystemUser) {
  return user.alias?.trim() || user.display_name?.trim() || user.username || `用户#${user.id}`
}

async function loadTodoSettings() {
  todoLoading.value = true
  try {
    const [settings, activeUsers] = await Promise.all([
      api.system.settings.getCustomerRateTodos(),
      api.system.users.list({ status: 1, page: 1, page_size: 1000 }),
    ])
    todoUserIDs.value = Array.isArray(settings?.customer_rate_user_ids)
      ? settings.customer_rate_user_ids.map(Number).filter(id => Number.isFinite(id) && id > 0)
      : []
    const options = new Map<number, { id: number; label: string }>()
    for (const user of activeUsers.items || []) options.set(user.id, { id: user.id, label: userLabel(user) })
    if (todoUserIDs.value.length) {
      const selectedUsers = await api.system.users.list({ ids: todoUserIDs.value.join(',') })
      for (const user of selectedUsers.items || []) options.set(user.id, { id: user.id, label: userLabel(user) })
    }
    todoUserOptions.value = Array.from(options.values())
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载待办提醒设置失败')
  } finally {
    todoLoading.value = false
  }
}

async function saveTodoSettings() {
  todoSaving.value = true
  try {
    const updated = await api.system.settings.updateCustomerRateTodos({ customer_rate_user_ids: todoUserIDs.value })
    todoUserIDs.value = updated.customer_rate_user_ids || []
    window.dispatchEvent(new Event('nfa:todos-refresh'))
    ElMessage.success('待办提醒设置已保存')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存待办提醒设置失败')
  } finally {
    todoSaving.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload: SystemTrafficSettings = {
      hide_non_settlement_schools_in_traffic: !!form.hide_non_settlement_schools_in_traffic,
      traffic_byte_unit_base: form.traffic_byte_unit_base,
      daily_traffic_volume_unit_base: form.daily_traffic_volume_unit_base,
      settlement_result_unit_base: form.settlement_result_unit_base,
      settlement_data_rate_unit: form.settlement_data_rate_unit,
      settlement_daily_detail_rate_unit: form.settlement_daily_detail_rate_unit,
      settlement_single_user_rate_unit: form.settlement_single_user_rate_unit,
    }
    const cfg = await api.system.settings.updateTraffic(payload)
    form.hide_non_settlement_schools_in_traffic = !!cfg?.hide_non_settlement_schools_in_traffic
    form.traffic_byte_unit_base = cfg?.traffic_byte_unit_base === 1000 ? 1000 : 1024
    form.daily_traffic_volume_unit_base = cfg?.daily_traffic_volume_unit_base === 1024 ? 1024 : 1000
    form.settlement_result_unit_base = cfg?.settlement_result_unit_base === 1000 ? 1000 : 1024
    form.settlement_data_rate_unit = cfg?.settlement_data_rate_unit === 'Gbps' ? 'Gbps' : 'Mbps'
    form.settlement_daily_detail_rate_unit = cfg?.settlement_daily_detail_rate_unit === 'Gbps' ? 'Gbps' : 'Mbps'
    form.settlement_single_user_rate_unit = cfg?.settlement_single_user_rate_unit === 'Mbps' ? 'Mbps' : 'Gbps'
    trafficSettings.apply(cfg)
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

.field-w-180 {
  width: 180px;
}

.todo-user-select {
  width: min(440px, 48%);
}

@media (max-width: 768px) {
  .todo-user-select {
    width: 100%;
  }
}
</style>
