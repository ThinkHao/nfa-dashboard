<template>
  <div class="settlement-config-tab">
    <div class="config-card">
      <div class="card-header">
        <h3>结算系统配置</h3>
        <div class="header-actions">
          <el-switch
            v-model="config.enabled"
            active-text="启用自动结算"
            inactive-text="禁用自动结算"
            @change="updateEnabledStatus"
          />
        </div>
      </div>

      <el-divider />

      <div v-loading="loading">
        <el-form
          ref="configForm"
          :model="config"
          label-width="120px"
          :disabled="!isEditing"
        >
          <el-form-item label="日结算时间">
            <el-time-picker
              v-model="config.daily_time"
              format="HH:mm"
              value-format="HH:mm"
              placeholder="选择时间"
              :disabled="!isEditing"
            />
            <div class="time-description">每天在此时间执行日结算任务，计算前一天的日95值</div>
          </el-form-item>

          <el-form-item label="周结算日">
            <el-select v-model="config.weekly_day" placeholder="选择星期">
              <el-option label="周一" :value="1" />
              <el-option label="周二" :value="2" />
              <el-option label="周三" :value="3" />
              <el-option label="周四" :value="4" />
              <el-option label="周五" :value="5" />
              <el-option label="周六" :value="6" />
              <el-option label="周日" :value="7" />
            </el-select>
            <div class="time-description">每周在此日期执行周结算任务，汇总上周的数据</div>
          </el-form-item>

          <el-form-item label="周结算时间">
            <el-time-picker
              v-model="config.weekly_time"
              format="HH:mm"
              value-format="HH:mm"
              placeholder="选择时间"
              :disabled="!isEditing"
            />
            <div class="time-description">在周结算日的此时间执行周结算任务</div>
          </el-form-item>

          <el-form-item label="上次执行时间">
            <div>{{ formatDateTime(config.last_execute_time) }}</div>
          </el-form-item>

          <el-form-item label="更新时间">
            <div>{{ formatDateTime(config.update_time) }}</div>
          </el-form-item>

          <el-form-item>
            <el-button
              v-if="!isEditing"
              type="primary"
              @click="startEditing"
            >
              编辑配置
            </el-button>
            <template v-else>
              <el-button type="primary" @click="saveConfig" :loading="saving">保存</el-button>
              <el-button @click="cancelEditing">取消</el-button>
            </template>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import api from '../../api'
import { ElMessage } from 'element-plus'
import type { SettlementConfig } from '../../types/settlement'

// 加载状态
const loading = ref(false)
const saving = ref(false)

// 编辑状态
const isEditing = ref(false)

// 配置数据
const config = reactive<SettlementConfig>({
  id: 0,
  daily_time: '02:00',
  weekly_day: 1,
  weekly_time: '02:00',
  enabled: true,
  last_execute_time: '',
  update_time: ''
})

// 时间选择器直接绑定字符串（value-format="HH:mm"）

// 获取结算配置
const fetchConfig = async () => {
  loading.value = true
  try {
    const response = await api.settlement.getConfig()
    if (response && typeof response === 'object') {
      Object.assign(config, response as any)
    }
  } catch (error) {
    console.error('获取结算配置失败', error)
    ElMessage.error('获取结算配置失败')
  } finally {
    loading.value = false
  }
}

// 更新启用状态
const updateEnabledStatus = async () => {
  saving.value = true
  try {
    await api.settlement.updateConfig(config)
    ElMessage.success(`${config.enabled ? '启用' : '禁用'}自动结算成功`)
  } catch (error) {
    console.error('更新结算配置失败', error)
    ElMessage.error('更新结算配置失败')
    // 恢复原来的状态
    config.enabled = !config.enabled
  } finally {
    saving.value = false
  }
}

// 开始编辑
const startEditing = () => {
  isEditing.value = true
}

// 取消编辑
const cancelEditing = () => {
  isEditing.value = false
  fetchConfig() // 重新获取配置，放弃修改
}

// 保存配置
const saveConfig = async () => {
  saving.value = true
  try {
    await api.settlement.updateConfig(config)
    ElMessage.success('保存配置成功')
    isEditing.value = false
    fetchConfig() // 重新获取最新配置
  } catch (error) {
    console.error('保存配置失败', error)
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

// 格式化日期时间
const formatDateTime = (dateTimeStr: string) => {
  if (!dateTimeStr || dateTimeStr === '0001-01-01T00:00:00Z') {
    return '未执行'
  }
  const date = new Date(dateTimeStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

// 组件挂载时获取数据
onMounted(() => {
  fetchConfig()
})
</script>

<style scoped>
.settlement-config-tab {
  padding: 10px;
}

.config-card {
  background-color: #fff;
  padding: 20px;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.card-header h3 {
  margin: 0;
}

.time-description {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
}
</style>
