<template>
  <div class="filter-rules-view">
    <h1 class="page-title">过滤规则管理</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button @click="goBack">返回</el-button>
            <span class="card-title">筛选</span>
          </div>
          <div>
            <el-button v-if="canWrite" type="primary" @click="openDialog()">新增规则</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="规则名">
          <el-input v-model="query.name" placeholder="按名称模糊查询" class="field-w-240" />
        </el-form-item>
        <el-form-item label="是否启用">
          <el-select v-model="query.enabled" clearable placeholder="全部" class="field-w-160">
            <el-option :value="true" label="启用" />
            <el-option :value="false" label="禁用" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
          <el-button @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header"><span class="card-title">规则列表</span></div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="name" label="规则名" min-width="180" show-overflow-tooltip />
        <el-table-column label="启用" width="100">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" :loading="row.__switching" @change="(val:boolean)=>onToggleEnabled(row, val)" />
          </template>
        </el-table-column>
        <el-table-column label="优先级" width="180">
          <template #default="{ row }">
            <el-input-number v-model="row.priority" :min="0" :max="999999" :step="1" />
            <el-button size="small" type="primary" :loading="row.__savingPriority" @click="onSavePriority(row)">保存</el-button>
          </template>
        </el-table-column>
        <el-table-column label="Region" min-width="160">
          <template #default="{ row }">{{ formatScope(row.scope_region) }}</template>
        </el-table-column>
        <el-table-column label="CP" min-width="140">
          <template #default="{ row }">{{ formatScope(row.scope_cp) }}</template>
        </el-table-column>
        <el-table-column label="学校名匹配" width="120">
          <template #default="{ row }">{{ formatMatchType(row.school_name_match_type) }}</template>
        </el-table-column>
        <el-table-column label="学校名称条件" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ formatScope(row.school_name_values) }}</template>
        </el-table-column>
        <el-table-column label="匹配数量" width="120">
          <template #default="{ row }">
            <el-tooltip :content="formatMatchedTooltip(row)" placement="top">
              <span>{{ Number(row.match_count || 0) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="canWrite" size="small" @click="openDialog(row)">编辑</el-button>
            <el-button v-if="canWrite" size="small" type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          background
          layout="prev, pager, next, sizes, total"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          :page-sizes="[10,20,50,100]"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑过滤规则' : '新增过滤规则'" width="760px">
      <el-form :model="form" label-width="140px" class="rule-form">
        <el-form-item label="规则名" required>
          <el-input v-model="form.name" placeholder="规则名称" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="0" :step="1" />
        </el-form-item>
        <el-form-item label="范围-Region">
          <el-select v-model="scopeRegion" multiple filterable :reserve-keyword="false" allow-create default-first-option clearable placeholder="如：华北、华南" class="field-w-full" />
          <div class="help">留空表示不限</div>
        </el-form-item>
        <el-form-item label="范围-CP">
          <el-select v-model="scopeCP" multiple filterable :reserve-keyword="false" allow-create default-first-option clearable placeholder="如：CT、CM" class="field-w-full" />
          <div class="help">留空表示不限</div>
        </el-form-item>
        <el-form-item label="学校名匹配方式">
          <el-radio-group v-model="form.school_name_match_type">
            <el-radio-button label="">不限</el-radio-button>
            <el-radio-button label="exact">精确</el-radio-button>
            <el-radio-button label="contains">包含</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="学校名称条件">
          <el-select v-model="schoolNameValues" multiple filterable :reserve-keyword="false" allow-create default-first-option clearable placeholder="输入学校名或关键词后回车" class="field-w-full" />
          <div class="help">留空表示不限；多值之间为“任一命中”。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { PaginatedData, FilterRule, CreateFilterRuleRequest, UpdateFilterRuleRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

type SchoolNameMatchType = '' | 'exact' | 'contains'

const auth = useAuthStore()
const router = useRouter()
const canWrite = computed(() => auth.hasPermission('rates.filter_rules.write'))

const loading = ref(false)
const items = ref<Array<FilterRule & { __switching?: boolean; __savingPriority?: boolean }>>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ name?: string; enabled?: boolean | null }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.name) p.name = query.name
  if (query.enabled !== undefined && query.enabled !== null) p.enabled = query.enabled
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<FilterRule> = await api.settlementRates.filterRules.list(buildParams())
    items.value = (res.items || []) as Array<FilterRule & { __switching?: boolean; __savingPriority?: boolean }>
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { name: undefined, enabled: undefined }); page.value = 1; pageSize.value = 10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

async function onToggleEnabled(row: any, val: boolean) {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  row.__switching = true
  try {
    await api.settlementRates.filterRules.setEnabled(row.id, val)
    row.enabled = val
    ElMessage.success('已更新')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '更新失败')
  } finally {
    row.__switching = false
  }
}

async function onSavePriority(row: any) {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  row.__savingPriority = true
  try {
    await api.settlementRates.filterRules.updatePriority(row.id, Number(row.priority) || 0)
    ElMessage.success('优先级已保存')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    row.__savingPriority = false
  }
}

const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref(false)
const editingId = ref<number | null>(null)
const originalEnabled = ref(true)
const originalPriority = ref(0)

const form = reactive<CreateFilterRuleRequest>({ name: '', enabled: true, priority: 0, school_name_match_type: '' })
const scopeRegion = ref<string[]>([])
const scopeCP = ref<string[]>([])
const schoolNameValues = ref<string[]>([])

function normalizeScopeList(v: any): string[] {
  if (Array.isArray(v)) return v.map((x) => String(x).trim()).filter(Boolean)
  if (typeof v === 'string') {
    const s = v.trim()
    return s ? [s] : []
  }
  return []
}

function openDialog(row?: FilterRule) {
  if (row) {
    editing.value = true
    editingId.value = row.id
    form.name = row.name
    form.enabled = !!row.enabled
    form.priority = row.priority
    form.school_name_match_type = row.school_name_match_type || ''
    originalEnabled.value = !!row.enabled
    originalPriority.value = row.priority
    scopeRegion.value = normalizeScopeList(row.scope_region)
    scopeCP.value = normalizeScopeList(row.scope_cp)
    schoolNameValues.value = normalizeScopeList(row.school_name_values)
  } else {
    editing.value = false
    editingId.value = null
    Object.assign(form, { name: '', enabled: true, priority: 0, school_name_match_type: '' })
    originalEnabled.value = true
    originalPriority.value = 0
    scopeRegion.value = []
    scopeCP.value = []
    schoolNameValues.value = []
  }
  dialogVisible.value = true
}

async function onSave() {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  if (!form.name?.trim()) { ElMessage.warning('规则名为必填'); return }
  if (schoolNameValues.value.length > 0 && !form.school_name_match_type) {
    ElMessage.warning('设置学校名称条件时，请选择匹配方式')
    return
  }

  const payloadBase = {
    name: form.name.trim(),
    scope_region: [...scopeRegion.value],
    scope_cp: [...scopeCP.value],
    school_name_match_type: (((schoolNameValues.value.length > 0 ? form.school_name_match_type : '') || '') as SchoolNameMatchType),
    school_name_values: [...schoolNameValues.value],
  }

  saving.value = true
  try {
    if (editing.value && editingId.value) {
      const updatePayload: UpdateFilterRuleRequest = { ...payloadBase }
      await api.settlementRates.filterRules.update(editingId.value, updatePayload)
      if (originalEnabled.value !== !!form.enabled) {
        await api.settlementRates.filterRules.setEnabled(editingId.value, !!form.enabled)
      }
      if (originalPriority.value !== (Number(form.priority) || 0)) {
        await api.settlementRates.filterRules.updatePriority(editingId.value, Number(form.priority) || 0)
      }
      ElMessage.success('更新成功')
    } else {
      const createPayload: CreateFilterRuleRequest = {
        enabled: !!form.enabled,
        priority: Number(form.priority) || 0,
        ...payloadBase,
      }
      await api.settlementRates.filterRules.create(createPayload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function onDelete(row: FilterRule) {
  try {
    await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '删除确认', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
  } catch { return }
  try {
    await api.settlementRates.filterRules.remove(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

function formatScope(v: any): string {
  const arr = normalizeScopeList(v)
  return arr.length > 0 ? arr.join('、') : '不限'
}

function formatMatchType(v?: string): string {
  if (v === 'exact') return '精确'
  if (v === 'contains') return '包含'
  return '不限'
}

function formatMatchedTooltip(row: FilterRule): string {
  const names = Array.isArray(row.matched_school_names) ? row.matched_school_names.filter(Boolean) : []
  if (names.length === 0) return '当前无匹配院校'
  const count = Number(row.match_count || names.length)
  return names.length < count ? `${names.join('、')} 等 ${count - names.length} 所` : names.join('、')
}

function goBack() {
  try {
    if (window.history.length > 1) {
      router.back()
      return
    }
  } catch {}
  router.push({ name: 'settlement-rates-customer' })
}

onMounted(() => { fetchData() })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-left { display: flex; align-items: center; gap: 8px; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.help { color: var(--text-muted); font-size: 12px; }
</style>
