<template>
  <div class="sync-rules-view">
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button @click="goBack">返回</el-button>
            <span class="card-title">同步规则管理</span>
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
        <div class="card-header"><span class="card-title">规则列表</span></div>
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
        <el-form-item label="范围-Region">
          <el-select v-model="scopeRegion" multiple filterable :reserve-keyword="false" allow-create default-first-option clearable placeholder="如：华北、华南" style="width: 100%" />
          <div class="help">留空表示不限</div>
        </el-form-item>
        <el-form-item label="范围-CP">
          <el-select v-model="scopeCP" multiple filterable :reserve-keyword="false" allow-create default-first-option clearable placeholder="如：CT、CM" style="width: 100%" />
          <div class="help">留空表示不限</div>
        </el-form-item>

        <el-form-item label="更新字段">
          <el-radio-group v-model="fieldsUpdateMode">
            <el-radio-button label="simple">简单</el-radio-button>
            <el-radio-button label="json">JSON</el-radio-button>
          </el-radio-group>
          <div v-if="fieldsUpdateMode==='simple'" class="kv-list">
            <div v-for="(row, idx) in kvRows" :key="idx" class="kv-row">
              <el-input v-model="row.key" placeholder="键（保存在 extra 下，如 remark）" style="width: 220px" />
              <el-input v-model="row.value" placeholder="值" style="width: 260px; margin-left: 8px;" />
              <el-button link type="danger" @click="removeKv(idx)">删除</el-button>
            </div>
            <el-button size="small" @click="addKv">新增一行</el-button>
            <div class="help">将保存到 fields_to_update.extra 下</div>
          </div>
          <div v-else>
            <el-input v-model="fieldsToUpdateText" type="textarea" :rows="4" placeholder='例如 {"extra":{"remark":"批量"}} 或 空' />
          </div>
        </el-form-item>

        <el-form-item label="动作" required>
          <el-radio-group v-model="actionMode">
            <el-radio-button label="template">模板</el-radio-button>
            <el-radio-button label="expr">表达式</el-radio-button>
            <el-radio-button label="json">JSON</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="actionMode==='template'" label="模板值">
          <div class="template-grid">
            <el-input-number v-model="templateValues.customer_fee" :step="0.01" :min="0" placeholder="customer_fee" />
            <el-input-number v-model="templateValues.network_line_fee" :step="0.01" :min="0" placeholder="network_line_fee" />
            <el-input-number v-model="templateValues.general_fee" :step="0.01" :min="0" placeholder="general_fee" />
          </div>
          <div class="help">留空的字段将不会写入</div>
        </el-form-item>

        <el-form-item v-if="actionMode==='expr'" label="表达式">
          <el-input v-model="exprText" type="textarea" :rows="4" placeholder="例如：customer_fee = base_fee + 0.02; network_line_fee = 0.12" />
        </el-form-item>

        <el-form-item v-if="actionMode==='json'" label="动作(JSON)" required>
          <el-input v-model="actionsText" type="textarea" :rows="6" placeholder='如 {"type":"template","values":{}} 或 {"type":"expr","expr":"..."}' />
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
const originalEnabled = ref<boolean>(true)
const originalPriority = ref<number>(0)

const form = reactive<CreateSyncRuleRequest>({ name: '', enabled: true, priority: 0, overwrite_strategy: 'always', actions: {} as any, condition_expr: undefined, scope_region: undefined, scope_cp: undefined, fields_to_update: undefined })
// 范围（表单化）
const scopeRegion = ref<string[]>([])
const scopeCP = ref<string[]>([])
// 更新字段：simple/json 两种模式
const fieldsUpdateMode = ref<'simple' | 'json'>('simple')
const kvRows = ref<{ key: string; value: string }[]>([])
const fieldsToUpdateText = ref('') // 仅 json 模式使用
// 动作：template/expr/json 三种模式
const actionMode = ref<'template' | 'expr' | 'json'>('template')
const templateValues = reactive<{ customer_fee?: number | null; network_line_fee?: number | null; general_fee?: number | null }>({})
const exprText = ref('')
const actionsText = ref('') // 仅 json 模式使用

function openDialog(row?: SyncRule) {
  if (row) {
    editing.value = true
    editingId.value = row.id
    form.name = row.name
    form.enabled = !!row.enabled
    originalEnabled.value = !!row.enabled
    form.priority = row.priority
    originalPriority.value = row.priority
    form.overwrite_strategy = row.overwrite_strategy
    form.condition_expr = row.condition_expr || undefined
    // 作用范围
    scopeRegion.value = Array.isArray(row.scope_region) ? [...(row.scope_region as string[])] : []
    scopeCP.value = Array.isArray(row.scope_cp) ? [...(row.scope_cp as string[])] : []

    // 更新字段
    if (row.fields_to_update && typeof row.fields_to_update === 'object') {
      const ft = row.fields_to_update as any
      if (ft.extra && typeof ft.extra === 'object') {
        fieldsUpdateMode.value = 'simple'
        kvRows.value = Object.entries(ft.extra).map(([key, value]) => ({ key: String(key), value: String(value as any) }))
        fieldsToUpdateText.value = ''
      } else {
        fieldsUpdateMode.value = 'json'
        fieldsToUpdateText.value = JSON.stringify(row.fields_to_update, null, 2)
        kvRows.value = []
      }
    } else {
      fieldsUpdateMode.value = 'simple'
      kvRows.value = []
      fieldsToUpdateText.value = ''
    }

    // 动作
    actionMode.value = 'json'
    actionsText.value = ''
    exprText.value = ''
    templateValues.customer_fee = null
    templateValues.network_line_fee = null
    templateValues.general_fee = null
    const act = row.actions as any
    if (act && typeof act === 'object' && typeof act.type === 'string') {
      if (act.type === 'template') {
        actionMode.value = 'template'
        const v = (act.values || {}) as any
        templateValues.customer_fee = v.customer_fee ?? null
        templateValues.network_line_fee = v.network_line_fee ?? null
        templateValues.general_fee = v.general_fee ?? null
      } else if (act.type === 'expr') {
        actionMode.value = 'expr'
        exprText.value = String(act.expr || '')
      } else {
        actionMode.value = 'json'
        actionsText.value = JSON.stringify(row.actions, null, 2)
      }
    } else if (row.actions) {
      actionMode.value = 'json'
      actionsText.value = JSON.stringify(row.actions, null, 2)
    } else {
      actionMode.value = 'template'
    }
  } else {
    editing.value = false
    editingId.value = null
    Object.assign(form, { name: '', enabled: true, priority: 0, overwrite_strategy: 'always', condition_expr: undefined })
    originalEnabled.value = !!form.enabled
    originalPriority.value = Number(form.priority) || 0
    scopeRegion.value = []
    scopeCP.value = []
    fieldsUpdateMode.value = 'simple'
    kvRows.value = []
    fieldsToUpdateText.value = ''
    actionMode.value = 'template'
    templateValues.customer_fee = null
    templateValues.network_line_fee = null
    templateValues.general_fee = null
    exprText.value = ''
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

  // 组装范围
  const scope_region = scopeRegion.value.length ? [...scopeRegion.value] : undefined
  const scope_cp = scopeCP.value.length ? [...scopeCP.value] : undefined

  // 组装更新字段
  let fields_to_update: any | undefined
  if (fieldsUpdateMode.value === 'json') {
    try {
      fields_to_update = safeParse(fieldsToUpdateText.value)
    } catch (e: any) {
      ElMessage.error(e?.message || '更新字段 JSON 格式错误')
      return
    }
  } else {
    const extraEntries = kvRows.value.filter(r => r.key && r.key.trim().length)
    if (extraEntries.length) {
      fields_to_update = { extra: Object.fromEntries(extraEntries.map(r => [r.key.trim(), r.value])) }
    }
  }

  // 组装动作
  let actions: any
  if (actionMode.value === 'template') {
    const values: any = {}
    if (templateValues.customer_fee != null) values.customer_fee = Number(templateValues.customer_fee)
    if (templateValues.network_line_fee != null) values.network_line_fee = Number(templateValues.network_line_fee)
    if (templateValues.general_fee != null) values.general_fee = Number(templateValues.general_fee)
    if (Object.keys(values).length === 0) { ElMessage.warning('请至少填写一项模板值'); return }
    actions = { type: 'template', values }
  } else if (actionMode.value === 'expr') {
    if (!exprText.value?.trim()) { ElMessage.warning('请填写表达式'); return }
    actions = { type: 'expr', expr: exprText.value }
  } else {
    if (!actionsText.value?.trim()) { ElMessage.warning('请填写动作 JSON'); return }
    try { actions = safeParse(actionsText.value) } catch (e: any) { ElMessage.error(e?.message || '动作 JSON 格式错误'); return }
  }

  const payloadBase = {
    name: form.name,
    overwrite_strategy: form.overwrite_strategy,
    condition_expr: form.condition_expr || undefined,
    scope_region,
    scope_cp,
    fields_to_update,
    actions,
  }

  saving.value = true
  try {
    if (editing.value && editingId.value) {
      // 后端不允许在 update 接口修改 enabled，需使用独立的 setEnabled 接口
      await api.settlementRates.syncRules.update(editingId.value, payloadBase as UpdateSyncRuleRequest)
      if (originalEnabled.value !== !!form.enabled) {
        await api.settlementRates.syncRules.setEnabled(editingId.value, !!form.enabled)
      }
      if (originalPriority.value !== (Number(form.priority) || 0)) {
        await api.settlementRates.syncRules.updatePriority(editingId.value, Number(form.priority) || 0)
      }
      ElMessage.success('更新成功')
    } else {
      const createPayload: CreateSyncRuleRequest = { enabled: !!form.enabled, priority: Number(form.priority) || 0, ...payloadBase }
      await api.settlementRates.syncRules.create(createPayload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { saving.value = false }
}

function addKv() { kvRows.value.push({ key: '', value: '' }) }
function removeKv(idx: number) { kvRows.value.splice(idx, 1) }

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
.filter-form { row-gap: 8px; }
.rule-form :deep(textarea) { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
