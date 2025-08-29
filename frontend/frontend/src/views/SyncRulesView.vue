<template>
  <div class="sync-rules-view">
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button @click="goBack">返回</el-button>
            <span class="title">同步规则管理</span>
          </div>
          <div>
            <el-button v-if="canWrite" type="primary" @click="openDialog()">新增规则</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="规则名">
          <el-input v-model="query.name" placeholder="按名称模糊查询" style="width: 240px" />
        </el-form-item>
        <el-form-item label="是否启用">
          <el-select v-model="query.enabled" clearable placeholder="全部" style="width: 160px">
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

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header"><span>规则列表</span></div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="name" label="规则名" min-width="180" show-overflow-tooltip />
        <el-table-column label="启用" width="120">
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
        <el-table-column prop="overwrite_strategy" label="覆盖策略" width="140" />
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

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑同步规则' : '新增同步规则'" width="800px">
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
        <el-form-item label="覆盖策略" required>
          <el-select v-model="form.overwrite_strategy" allow-create filterable default-first-option placeholder="选择或输入">
            <el-option label="always" value="always" />
            <el-option label="if_empty" value="if_empty" />
          </el-select>
        </el-form-item>
        <el-form-item label="条件表达式">
          <el-input v-model="form.condition_expr" placeholder="可选：如 region == '华北' && cp in ['CT','CM']" />
        </el-form-item>
        <el-form-item label="范围-Region(JSON)">
          <el-input v-model="scopeRegionText" type="textarea" :rows="3" placeholder='如 ["华北","华南"] 或 空' />
        </el-form-item>
        <el-form-item label="范围-CP(JSON)">
          <el-input v-model="scopeCPText" type="textarea" :rows="3" placeholder='如 ["CT","CM"] 或 空' />
        </el-form-item>
        <el-form-item label="更新字段(JSON)">
          <el-input v-model="fieldsToUpdateText" type="textarea" :rows="4" placeholder='如 {"extra":{"k":"v"}} 或 {"customer_fee": 1.23} 或 空' />
        </el-form-item>
        <el-form-item label="动作(JSON)" required>
          <el-input v-model="actionsText" type="textarea" :rows="6" placeholder='必填，如 {"type":"template","values":{}} 或 {"type":"expr","expr":"..."}' />
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
import type { PaginatedData, SyncRule, CreateSyncRuleRequest, UpdateSyncRuleRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canWrite = computed(() => auth.hasPermission('rates.sync_rules.write'))

const loading = ref(false)
const items = ref<SyncRule[]>([])
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
    const res: PaginatedData<SyncRule> = await api.settlementRates.syncRules.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally { loading.value = false }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { name: undefined, enabled: undefined }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

async function onToggleEnabled(row: any, val: boolean) {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  row.__switching = true
  try {
    await api.settlementRates.syncRules.setEnabled(row.id, val)
    row.enabled = val
    ElMessage.success('已更新')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '更新失败')
  } finally { row.__switching = false }
}

async function onSavePriority(row: any) {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  row.__savingPriority = true
  try {
    await api.settlementRates.syncRules.updatePriority(row.id, Number(row.priority) || 0)
    ElMessage.success('优先级已保存')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { row.__savingPriority = false }
}

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref(false)
const editingId = ref<number | null>(null)

const form = reactive<CreateSyncRuleRequest>({ name: '', enabled: true, priority: 0, overwrite_strategy: 'always', actions: {} as any, condition_expr: undefined, scope_region: undefined, scope_cp: undefined, fields_to_update: undefined })
const scopeRegionText = ref('')
const scopeCPText = ref('')
const fieldsToUpdateText = ref('')
const actionsText = ref('')

function openDialog(row?: SyncRule) {
  if (row) {
    editing.value = true
    editingId.value = row.id
    form.name = row.name
    form.enabled = !!row.enabled
    form.priority = row.priority
    form.overwrite_strategy = row.overwrite_strategy
    form.condition_expr = row.condition_expr || undefined
    scopeRegionText.value = row.scope_region ? JSON.stringify(row.scope_region, null, 2) : ''
    scopeCPText.value = row.scope_cp ? JSON.stringify(row.scope_cp, null, 2) : ''
    fieldsToUpdateText.value = row.fields_to_update ? JSON.stringify(row.fields_to_update, null, 2) : ''
    actionsText.value = row.actions ? JSON.stringify(row.actions, null, 2) : ''
  } else {
    editing.value = false
    editingId.value = null
    Object.assign(form, { name: '', enabled: true, priority: 0, overwrite_strategy: 'always', condition_expr: undefined })
    scopeRegionText.value = ''
    scopeCPText.value = ''
    fieldsToUpdateText.value = ''
    actionsText.value = ''
  }
  dialogVisible.value = true
}

function safeParse(text: string): any | undefined {
  if (!text || !text.trim()) return undefined
  try { return JSON.parse(text) } catch { throw new Error('JSON 解析失败') }
}

async function onSave() {
  if (!canWrite.value) { ElMessage.warning('无写权限'); return }
  if (!form.name?.trim()) { ElMessage.warning('规则名为必填'); return }
  if (!form.overwrite_strategy?.trim()) { ElMessage.warning('覆盖策略为必填'); return }
  if (!actionsText.value?.trim()) { ElMessage.warning('动作(JSON)为必填'); return }

  let payload: CreateSyncRuleRequest | UpdateSyncRuleRequest
  try {
    const scope_region = safeParse(scopeRegionText.value)
    const scope_cp = safeParse(scopeCPText.value)
    const fields_to_update = safeParse(fieldsToUpdateText.value)
    const actions = safeParse(actionsText.value)
    payload = {
      name: form.name,
      enabled: !!form.enabled,
      priority: Number(form.priority) || 0,
      overwrite_strategy: form.overwrite_strategy,
      condition_expr: form.condition_expr || undefined,
      scope_region,
      scope_cp,
      fields_to_update,
      actions: actions as any,
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '请检查 JSON 字段格式')
    return
  }

  saving.value = true
  try {
    if (editing.value && editingId.value) {
      await api.settlementRates.syncRules.update(editingId.value, payload as UpdateSyncRuleRequest)
      ElMessage.success('更新成功')
    } else {
      await api.settlementRates.syncRules.create(payload as CreateSyncRuleRequest)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { saving.value = false }
}

async function onDelete(row: SyncRule) {
  try {
    await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '删除确认', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
  } catch { return }
  try {
    await api.settlementRates.syncRules.remove(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

onMounted(() => { fetchData() })

function goBack() {
  try {
    // 有前置历史就返回
    if (window.history.length > 1) {
      router.back()
      return
    }
  } catch {}
  // 无历史时回到客户费率列表
  router.push({ name: 'settlement-rates-customer' })
}
</script>

<style scoped>
.sync-rules-view { padding: 20px; }
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-left { display: flex; align-items: center; gap: 8px; }
.title { font-weight: 600; }
.filter-form { row-gap: 8px; }
.rule-form :deep(textarea) { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
