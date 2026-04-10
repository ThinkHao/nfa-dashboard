<template>
  <div class="page-container">
    <PageHeader title="流量范围管理" description="为用户配置流量监控的可见范围，不授予账号管理能力。" />

    <FilterPanel>
      <div class="card-header">
        <span class="card-title">选择用户</span>
        <div>
          <el-button type="primary" :disabled="!selectedUserId" :loading="saving" @click="saveRules">保存规则</el-button>
          <el-button :disabled="!selectedUserId" @click="reloadCurrent">刷新</el-button>
        </div>
      </div>
      <el-form inline label-width="90px" class="filter-form">
        <el-form-item label="目标用户">
          <el-select
            v-model="selectedUserId"
            class="field-lg"
            filterable
            remote
            reserve-keyword
            clearable
            placeholder="输入用户名或别名搜索"
            :remote-method="searchUsers"
            :loading="userLoading"
            @change="handleUserChange"
          >
            <el-option
              v-for="user in users"
              :key="user.id"
              :label="`${user.display_name} (${user.username})`"
              :value="user.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
    </FilterPanel>

    <SectionCard title="范围规则">
      <template #actions>
        <el-button type="primary" plain :disabled="!selectedUserId" @click="addRuleGroup">新增规则组</el-button>
      </template>

      <div v-loading="rulesLoading">
        <div v-if="selectedUserId && ruleGroups.length > 0" class="rule-groups">
          <div v-for="(group, groupIndex) in ruleGroups" :key="group.id ?? `group-${groupIndex}`" class="rule-group">
            <div class="rule-group-header">
              <div class="rule-group-title">规则组 {{ groupIndex + 1 }}</div>
              <div class="rule-group-actions">
                <el-select v-model="group.rule_type" class="rule-type-field">
                  <el-option label="允许" value="allow" />
                  <el-option label="拒绝" value="deny" />
                </el-select>
                <el-button type="primary" link @click="addCondition(groupIndex)">新增条件</el-button>
                <el-button type="danger" link @click="removeRuleGroup(groupIndex)">删除规则组</el-button>
              </div>
            </div>

            <el-table :data="group.conditions" border stripe>
              <el-table-column label="维度" width="160">
                <template #default="{ row }">
                  <el-select v-model="row.dimension_type" class="cell-field" @change="handleDimensionChange(row)">
                    <el-option label="区域" value="region" />
                    <el-option label="CP" value="cp" />
                    <el-option label="院校" value="school" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="取值" min-width="280">
                <template #default="{ row, $index }">
                  <el-select
                    v-if="row.dimension_type !== 'school'"
                    v-model="row.dimension_value"
                    class="cell-field"
                    filterable
                    clearable
                    :placeholder="row.dimension_type === 'region' ? '请选择区域' : '请选择 CP'"
                  >
                    <el-option
                      v-for="option in row.dimension_type === 'region' ? regionOptions : cpOptions"
                      :key="`${row.dimension_type}-${option.value}`"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                  <el-select
                    v-else
                    v-model="row.dimension_value"
                    class="cell-field"
                    filterable
                    remote
                    reserve-keyword
                    clearable
                    :remote-method="(query) => searchSchoolOptions(groupIndex, $index, query)"
                    :loading="schoolOptionsLoading[getConditionKey(groupIndex, $index)]"
                    placeholder="输入院校名称搜索"
                    @visible-change="(visible) => visible && searchSchoolOptions(groupIndex, $index, '')"
                  >
                    <el-option
                      v-for="option in schoolOptionsMap[getConditionKey(groupIndex, $index)] || []"
                      :key="`${option.school_id}-${option.region}-${option.cp}`"
                      :label="formatTrafficScopeSchoolOptionLabel(option)"
                      :value="option.value"
                    />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ $index }">
                  <el-button type="danger" link @click="removeCondition(groupIndex, $index)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>

        <el-empty
          v-if="!rulesLoading && selectedUserId && ruleGroups.length === 0"
          description="当前没有配置规则组，将回退到默认范围策略或旧院校绑定。"
        />
        <el-empty v-if="!selectedUserId" description="请先选择用户" />
      </div>
    </SectionCard>

    <SectionCard title="生效预览">
      <div class="preview-grid" v-loading="previewLoading">
        <div class="preview-block">
          <div class="preview-label">当前来源</div>
          <el-tag :type="sourceTagType(preview.source)">{{ sourceLabel(preview.source) }}</el-tag>
        </div>
        <div class="preview-block">
          <div class="preview-label">旧绑定院校数</div>
          <div class="preview-value">{{ preview.legacy_school_ids.length }}</div>
        </div>
        <div class="preview-block">
          <div class="preview-label">最终可见院校数</div>
          <div class="preview-value">{{ preview.allowed_schools.length }}</div>
        </div>
      </div>

      <el-alert
        v-if="selectedUserId && preview.source === 'none'"
        type="warning"
        :closable="false"
        title="该用户当前没有任何流量监控数据范围，进入流量监控页后将看不到院校数据。"
        class="mt-12"
      />

      <el-descriptions v-if="selectedUserId" :column="1" border class="mt-12">
        <el-descriptions-item label="旧绑定院校 ID">
          {{ preview.legacy_school_ids.length ? preview.legacy_school_ids.join('，') : '无' }}
        </el-descriptions-item>
      </el-descriptions>

      <el-table :data="preview.allowed_schools" border stripe class="mt-12">
        <el-table-column prop="school_id" label="院校ID" min-width="180" />
        <el-table-column prop="school_name" label="院校名称" min-width="220" />
        <el-table-column prop="region" label="区域" width="140" />
        <el-table-column prop="cp" label="CP" width="140" />
      </el-table>
    </SectionCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type {
  TrafficScopeCondition,
  TrafficScopeOptionItem,
  TrafficScopePreview,
  TrafficScopeRuleGroup,
  TrafficScopeUserLite,
} from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterPanel from '@/components/ui/FilterPanel.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import {
  createEmptyTrafficScopeCondition,
  createEmptyTrafficScopeRuleGroup,
  normalizeTrafficScopeRuleGroups,
} from './traffic-scope-rule-groups'
import {
  buildTrafficScopeOptionRequest,
  formatTrafficScopeSchoolOptionLabel,
} from './traffic-scope-options'

const users = ref<TrafficScopeUserLite[]>([])
const userLoading = ref(false)
const rulesLoading = ref(false)
const previewLoading = ref(false)
const saving = ref(false)
const selectedUserId = ref<number | undefined>()
const ruleGroups = ref<TrafficScopeRuleGroup[]>([])
const regionOptions = ref<TrafficScopeOptionItem[]>([])
const cpOptions = ref<TrafficScopeOptionItem[]>([])
const schoolOptionsMap = ref<Record<string, TrafficScopeOptionItem[]>>({})
const schoolOptionsLoading = reactive<Record<string, boolean>>({})
const preview = reactive<TrafficScopePreview>({
  user_id: 0,
  source: 'none',
  rules: [],
  legacy_school_ids: [],
  allowed_schools: [],
})

async function searchUsers(keyword = '') {
  userLoading.value = true
  try {
    const res = await api.system.trafficScopes.listUsers({ username: keyword || undefined, page: 1, page_size: 20 })
    users.value = res.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载用户失败')
  } finally {
    userLoading.value = false
  }
}

function getConditionKey(groupIndex: number, conditionIndex: number) {
  return `${groupIndex}-${conditionIndex}`
}

async function loadStaticOptions() {
  try {
    const [regions, cps] = await Promise.all([
      api.system.trafficScopes.options({ dimension: 'region', limit: 200 }),
      api.system.trafficScopes.options({ dimension: 'cp', limit: 200 }),
    ])
    regionOptions.value = regions.items || []
    cpOptions.value = cps.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载范围选项失败')
  }
}

function handleDimensionChange(row: TrafficScopeCondition) {
  row.dimension_value = ''
}

async function searchSchoolOptions(groupIndex: number, conditionIndex: number, keyword = '') {
  const key = getConditionKey(groupIndex, conditionIndex)
  const group = ruleGroups.value[groupIndex]
  if (!group) return
  schoolOptionsLoading[key] = true
  try {
    const params = buildTrafficScopeOptionRequest('school', group.conditions, keyword)
    const res = await api.system.trafficScopes.options(params)
    schoolOptionsMap.value[key] = res.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载院校选项失败')
    schoolOptionsMap.value[key] = []
  } finally {
    schoolOptionsLoading[key] = false
  }
}

function addRuleGroup() {
  ruleGroups.value.push(createEmptyTrafficScopeRuleGroup())
}

function removeRuleGroup(index: number) {
  ruleGroups.value.splice(index, 1)
}

function addCondition(groupIndex: number) {
  ruleGroups.value[groupIndex]?.conditions.push(createEmptyTrafficScopeCondition())
}

function removeCondition(groupIndex: number, conditionIndex: number) {
  ruleGroups.value[groupIndex]?.conditions.splice(conditionIndex, 1)
}

function resetPreview() {
  preview.user_id = 0
  preview.source = 'none'
  preview.rules = []
  preview.legacy_school_ids = []
  preview.allowed_schools = []
}

async function loadRules(userId: number) {
  rulesLoading.value = true
  try {
    const res = await api.system.trafficScopes.list(userId)
    ruleGroups.value = (res.items || []).map((item) => ({
      id: item.id,
      user_id: item.user_id,
      rule_type: item.rule_type,
      created_at: item.created_at,
      updated_at: item.updated_at,
      conditions: (item.conditions || []).map((condition) => ({
        id: condition.id,
        dimension_type: condition.dimension_type,
        dimension_value: condition.dimension_value,
        created_at: condition.created_at,
      })),
    }))
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载规则失败')
    ruleGroups.value = []
  } finally {
    rulesLoading.value = false
  }
}

async function loadPreview(userId: number) {
  previewLoading.value = true
  try {
    const res = await api.system.trafficScopes.preview(userId)
    preview.user_id = res.user_id
    preview.source = res.source
    preview.rules = res.rules || []
    preview.legacy_school_ids = res.legacy_school_ids || []
    preview.allowed_schools = res.allowed_schools || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载预览失败')
    resetPreview()
  } finally {
    previewLoading.value = false
  }
}

async function handleUserChange(userId?: number) {
  if (!userId) {
    ruleGroups.value = []
    resetPreview()
    return
  }
  await Promise.all([loadRules(userId), loadPreview(userId)])
}

async function reloadCurrent() {
  if (!selectedUserId.value) return
  await handleUserChange(selectedUserId.value)
}

async function saveRules() {
  if (!selectedUserId.value) {
    ElMessage.warning('请先选择用户')
    return
  }
  const payload = normalizeTrafficScopeRuleGroups(ruleGroups.value)
  saving.value = true
  try {
    await api.system.trafficScopes.replace(selectedUserId.value, payload)
    ElMessage.success('保存成功')
    await handleUserChange(selectedUserId.value)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function sourceLabel(source: TrafficScopePreview['source']) {
  switch (source) {
    case 'policy_rule':
      return '新规则'
    case 'legacy_user_school':
      return '旧院校绑定'
    case 'default_admin_role':
      return '管理员默认全量'
    default:
      return '无范围'
  }
}

function sourceTagType(source: TrafficScopePreview['source']) {
  switch (source) {
    case 'policy_rule':
      return 'success'
    case 'legacy_user_school':
      return 'warning'
    case 'default_admin_role':
      return 'danger'
    default:
      return 'info'
  }
}

onMounted(() => {
  searchUsers()
  loadStaticOptions()
})
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.field-lg:deep(.el-select__wrapper) { width: 360px !important; }
.cell-field:deep(.el-select__wrapper) { width: 100% !important; }
.rule-type-field:deep(.el-select__wrapper) { width: 120px !important; }
.rule-groups {
  display: grid;
  gap: 16px;
}
.rule-group {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  padding: 16px;
}
.rule-group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.rule-group-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-default);
}
.rule-group-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.preview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.preview-block {
  padding: 14px 16px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
}
.preview-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.preview-value {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-default);
}
.mt-12 { margin-top: 12px; }

@media (max-width: 900px) {
  .preview-grid { grid-template-columns: 1fr; }
  .rule-group-header { align-items: flex-start; flex-direction: column; }
}
</style>
