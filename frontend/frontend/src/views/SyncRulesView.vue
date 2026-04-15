<template>
  <div class="sync-rules-view">
    <div class="page-heading">
      <div>
        <h1 class="page-title">同步规则管理</h1>
        <p class="page-subtitle">按同步规则批量生成和更新客户业务费率。</p>
      </div>
    </div>

    <el-card shadow="never" class="box-card filter-panel">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button class="back-button" @click="goBack">返回</el-button>
            <div>
              <span class="card-title">筛选</span>
              <p class="card-subtitle">快速定位同步规则并继续维护。</p>
            </div>
          </div>
          <div class="header-actions">
            <el-button v-if="canWrite" type="primary" @click="openDialog()">新增规则</el-button>
          </div>
        </div>
      </template>

      <el-form :model="query" label-position="top" class="filter-form">
        <div class="filter-grid">
          <el-form-item label="规则名" class="filter-item">
            <el-input v-model="query.name" placeholder="按名称模糊查询" />
          </el-form-item>
          <el-form-item label="是否启用" class="filter-item filter-item-small">
            <el-select v-model="query.enabled" clearable placeholder="全部">
              <el-option :value="true" label="启用" />
              <el-option :value="false" label="禁用" />
            </el-select>
          </el-form-item>
          <div class="filter-actions">
            <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
            <el-button @click="onReset">重置</el-button>
          </div>
        </div>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2 list-panel">
      <template #header>
        <div class="card-header">
          <div>
            <span class="card-title">规则列表</span>
            <p class="card-subtitle">优先级越靠前，规则越早参与同步执行。</p>
          </div>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="name" label="规则名" min-width="180" show-overflow-tooltip />
        <el-table-column label="启用" width="120">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" :loading="row.__switching" @change="(val:boolean)=>onToggleEnabled(row, val)" />
          </template>
        </el-table-column>
        <el-table-column label="优先级" width="160">
          <template #default="{ row }">
            <div class="priority-cell">
              <el-input-number v-model="row.priority" :min="0" :max="999999" :step="1" controls-position="right" />
              <el-button size="small" type="primary" plain :loading="row.__savingPriority" @click="onSavePriority(row)">保存</el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="overwrite_strategy" label="覆盖策略" width="140" />
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button v-if="canWrite" size="small" @click="openDialog(row)">编辑</el-button>
              <el-button v-if="canWrite" size="small" type="danger" plain @click="onDelete(row)">删除</el-button>
            </div>
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

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑同步规则' : '新增同步规则'" width="820px">
      <el-form :model="form" label-position="top" class="rule-form">
        <div class="dialog-section">
          <div class="dialog-section-title">基础信息</div>
          <div class="dialog-grid dialog-grid-basic">
            <el-form-item label="规则名" required class="dialog-item dialog-item-wide">
              <el-input v-model="form.name" placeholder="规则名称" />
            </el-form-item>
            <el-form-item label="启用" class="dialog-item">
              <el-switch v-model="form.enabled" />
            </el-form-item>
            <el-form-item label="优先级" class="dialog-item">
              <el-input-number v-model="form.priority" :min="0" :step="1" controls-position="right" />
            </el-form-item>
          </div>
          <div class="dialog-grid dialog-grid-basic dialog-grid-secondary">
            <el-form-item label="覆盖策略" required class="dialog-item">
              <el-select v-model="form.overwrite_strategy" allow-create filterable default-first-option placeholder="选择或输入">
                <el-option label="always" value="always" />
                <el-option label="if_empty" value="if_empty" />
              </el-select>
            </el-form-item>
            <el-form-item label="条件表达式" class="dialog-item dialog-item-wide">
              <el-input v-model="form.condition_expr" placeholder="可选：如 region == '华北' && cp in ['CT','CM']" />
            </el-form-item>
          </div>
        </div>

        <div class="dialog-section">
          <div class="dialog-section-title">匹配范围</div>
          <div class="dialog-grid">
            <el-form-item label="范围-Region" class="dialog-item">
              <el-select
                v-model="scopeRegion"
                multiple
                filterable
                :reserve-keyword="false"
                clearable
                :loading="optionsLoading"
                placeholder="选择 Region"
                class="field-w-full"
              >
                <el-option v-for="item in mergedRegionOptions" :key="item" :label="item" :value="item" />
              </el-select>
              <div class="help">留空表示不限</div>
            </el-form-item>
            <el-form-item label="范围-CP" class="dialog-item">
              <el-select
                v-model="scopeCP"
                multiple
                filterable
                :reserve-keyword="false"
                clearable
                :loading="optionsLoading"
                placeholder="选择 CP"
                class="field-w-full"
              >
                <el-option v-for="item in mergedCPOptions" :key="item" :label="item" :value="item" />
              </el-select>
              <div class="help">留空表示不限</div>
            </el-form-item>
          </div>
        </div>

        <div class="dialog-section">
          <div class="dialog-section-title">更新字段</div>
          <el-form-item label="更新方式">
            <el-radio-group v-model="fieldsUpdateMode" class="match-mode-group">
              <el-radio-button label="simple">简单</el-radio-button>
              <el-radio-button label="json">JSON</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <div v-if="fieldsUpdateMode==='simple'" class="kv-list">
            <div v-for="(row, idx) in kvRows" :key="idx" class="kv-row">
              <el-input v-model="row.key" placeholder="键（保存在 extra 下，如 remark）" class="field-w-220" />
              <el-input v-model="row.value" placeholder="值" class="field-w-260 ml-8" />
              <el-button link type="danger" @click="removeKv(idx)">删除</el-button>
            </div>
            <div class="kv-actions">
              <div class="help">将保存到 fields_to_update.extra 下</div>
              <el-button size="small" @click="addKv">新增一行</el-button>
            </div>
          </div>
          <div v-else class="textarea-block">
            <el-input v-model="fieldsToUpdateText" type="textarea" :rows="4" placeholder='例如 {"extra":{"remark":"批量"}} 或 空' />
          </div>
        </div>

        <div class="dialog-section">
          <div class="dialog-section-title">动作</div>
          <el-form-item label="动作类型" required>
            <el-radio-group v-model="actionMode" class="match-mode-group">
              <el-radio-button label="template">模板</el-radio-button>
              <el-radio-button label="expr">表达式</el-radio-button>
              <el-radio-button label="json">JSON</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="actionMode==='template'">
            <template #label>
              模板值
              <el-tooltip content="留空的字段将不会写入" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <div class="template-list">
              <div class="template-row">
                <el-input-number v-model="templateValues.customer_fee" :step="0.01" :min="0" controls-position="right" />
                <span class="row-hint">客户费率</span>
              </div>
              <div class="template-row">
                <el-input-number v-model="templateValues.network_line_fee" :step="0.01" :min="0" controls-position="right" />
                <span class="row-hint">线路费率</span>
              </div>
              <div class="template-row">
                <el-input-number v-model="templateValues.general_fee" :step="0.01" :min="0" controls-position="right" />
                <span class="row-hint">通用费率</span>
              </div>
              <div class="template-row">
                <el-input-number v-model="templateValues.channel_rate" :step="0.01" :min="0" controls-position="right" />
                <span class="row-hint">渠道费率</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item v-if="actionMode==='expr'" label="表达式">
            <el-input v-model="exprText" type="textarea" :rows="4" placeholder="例如：customer_fee = base_fee + 0.02; network_line_fee = 0.12" />
          </el-form-item>

          <el-form-item v-if="actionMode==='json'" label="动作(JSON)" required>
            <el-input v-model="actionsText" type="textarea" :rows="6" placeholder='如 {"type":"template","values":{}} 或 {"type":"expr","expr":"..."}' />
          </el-form-item>
        </div>
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
import { QuestionFilled } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { PaginatedData, SyncRule, CreateSyncRuleRequest, UpdateSyncRuleRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import { mergeScopeOptions } from './settlement-rule-options'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'

const auth = useAuthStore()
const router = useRouter()
const canWrite = computed(() => auth.hasPermission('rates.sync_rules.write'))

const loading = ref(false)
const items = ref<SyncRule[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const queryCtl = useCancelableQuery()

const query = reactive<{ name?: string; enabled?: boolean | null }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.name) p.name = query.name
  if (query.enabled !== undefined && query.enabled !== null) p.enabled = query.enabled
  return p
}

async function fetchData(signal?: AbortSignal) {
  loading.value = true
  try {
    const res: PaginatedData<SyncRule> = await api.settlementRates.syncRules.list(buildParams(), { signal })
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    if (isAbortError(e)) return
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally { loading.value = false }
}

function onSearch() { page.value = 1; queryCtl.run((signal) => fetchData(signal), { toggleIfRunning: true }) }
function onReset() { Object.assign(query, { name: undefined, enabled: undefined }); page.value=1; pageSize.value=10; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageChange(p: number) { page.value = p; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }

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
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { row.__savingPriority = false }
}

const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref(false)
const optionsLoading = ref(false)
const editingId = ref<number | null>(null)
const originalEnabled = ref<boolean>(true)
const originalPriority = ref<number>(0)
const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])

const form = reactive<CreateSyncRuleRequest>({ name: '', enabled: true, priority: 0, overwrite_strategy: 'always', actions: {} as any, condition_expr: undefined, scope_region: undefined, scope_cp: undefined, fields_to_update: undefined })
const scopeRegion = ref<string[]>([])
const scopeCP = ref<string[]>([])
const fieldsUpdateMode = ref<'simple' | 'json'>('simple')
const kvRows = ref<{ key: string; value: string }[]>([])
const fieldsToUpdateText = ref('')
const actionMode = ref<'template' | 'expr' | 'json'>('template')
const templateValues = reactive<{ customer_fee?: number | null; network_line_fee?: number | null; general_fee?: number | null; channel_rate?: number | null }>({})
const exprText = ref('')
const actionsText = ref('')
const mergedRegionOptions = computed(() => mergeScopeOptions(scopeRegion.value, regionOptions.value))
const mergedCPOptions = computed(() => mergeScopeOptions(scopeCP.value, cpOptions.value))

async function loadOptions() {
  optionsLoading.value = true
  try {
    const res = await api.settlementRates.syncRules.options()
    regionOptions.value = Array.isArray(res.regions) ? res.regions : []
    cpOptions.value = Array.isArray(res.cps) ? res.cps : []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载规则选项失败')
  } finally {
    optionsLoading.value = false
  }
}

function normalizeScopeList(v: any): string[] {
  if (Array.isArray(v)) {
    return v.map((x) => String(x).trim()).filter(Boolean)
  }
  if (typeof v === 'string') {
    const s = v.trim()
    return s ? [s] : []
  }
  return []
}

function openDialog(row?: SyncRule) {
  if (regionOptions.value.length === 0 && cpOptions.value.length === 0 && !optionsLoading.value) {
    void loadOptions()
  }
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
    scopeRegion.value = normalizeScopeList(row.scope_region)
    scopeCP.value = normalizeScopeList(row.scope_cp)

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

    actionMode.value = 'json'
    actionsText.value = ''
    exprText.value = ''
    templateValues.customer_fee = null
    templateValues.network_line_fee = null
    templateValues.general_fee = null
    templateValues.channel_rate = null
    const act = row.actions as any
    if (act && typeof act === 'object' && typeof act.type === 'string') {
      if (act.type === 'template') {
        actionMode.value = 'template'
        const v = (act.values || {}) as any
        templateValues.customer_fee = v.customer_fee ?? null
        templateValues.network_line_fee = v.network_line_fee ?? null
        templateValues.general_fee = v.general_fee ?? null
        templateValues.channel_rate = v.channel_rate ?? v.final_fee ?? null
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
    templateValues.channel_rate = null
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

  const scope_region = [...scopeRegion.value]
  const scope_cp = [...scopeCP.value]

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

  let actions: any
  if (actionMode.value === 'template') {
    const values: any = {}
    if (templateValues.customer_fee != null) values.customer_fee = Number(templateValues.customer_fee)
    if (templateValues.network_line_fee != null) values.network_line_fee = Number(templateValues.network_line_fee)
    if (templateValues.general_fee != null) values.general_fee = Number(templateValues.general_fee)
    if (templateValues.channel_rate != null) values.channel_rate = Number(templateValues.channel_rate)
    if (Object.keys(values).length === 0) { ElMessage.warning('请至少填写一项模板值'); return }
    actions = { type: 'template', values }
  } else if (actionMode.value === 'expr') {
    if (!exprText.value?.trim()) { ElMessage.warning('请填写表达式'); return }
    actions = { type: 'expr', expr: exprText.value }
  } else {
    if (!actionsText.value?.trim()) { ElMessage.warning('请填写动作 JSON'); return }
    try { actions = safeParse(actionsText.value) } catch (e: any) { ElMessage.error(e?.message || '动作 JSON 格式错误'); return }
  }

  const conditionExprTrimmed = (form.condition_expr || '').trim()

  const payloadBase = {
    name: form.name,
    overwrite_strategy: form.overwrite_strategy,
    scope_region,
    scope_cp,
    fields_to_update,
    actions,
  }

  saving.value = true
  try {
    if (editing.value && editingId.value) {
      const updatePayload: UpdateSyncRuleRequest = {
        ...payloadBase,
        condition_expr: conditionExprTrimmed,
      }
      await api.settlementRates.syncRules.update(editingId.value, updatePayload)
      if (originalEnabled.value !== !!form.enabled) {
        await api.settlementRates.syncRules.setEnabled(editingId.value, !!form.enabled)
      }
      if (originalPriority.value !== (Number(form.priority) || 0)) {
        await api.settlementRates.syncRules.updatePriority(editingId.value, Number(form.priority) || 0)
      }
      ElMessage.success('更新成功')
    } else {
      const createPayload: CreateSyncRuleRequest = {
        enabled: !!form.enabled,
        priority: Number(form.priority) || 0,
        ...payloadBase,
        condition_expr: conditionExprTrimmed || undefined,
      }
      await api.settlementRates.syncRules.create(createPayload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
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

onMounted(() => {
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  loadOptions()
})
usePageRefresh(() => {
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
})

function goBack() {
  try {
    if (window.history.length > 1) {
      router.back()
      return
    }
  } catch {}
  router.push({ name: 'settlement-rates-customer' })
}
</script>

<style scoped>
.page-heading {
  margin-bottom: 12px;
}

.page-subtitle,
.card-subtitle {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.5;
}

.box-card {
  margin-bottom: 12px;
}

.filter-panel,
.list-panel {
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.header-actions {
  display: flex;
  justify-content: flex-end;
  flex: 0 0 auto;
}

.back-button {
  flex: 0 0 auto;
}

.filter-form :deep(.el-form-item) {
  margin-bottom: 0;
}

.filter-grid {
  display: grid;
  grid-template-columns: minmax(220px, 1.6fr) minmax(160px, 0.8fr) auto;
  gap: 12px 16px;
  align-items: end;
}

.filter-item {
  min-width: 0;
}

.filter-item-small {
  max-width: 220px;
}

.filter-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-start;
  flex-wrap: wrap;
}

.priority-cell {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}

.priority-cell :deep(.el-input-number) {
  width: 100%;
}

.row-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.rule-form {
  padding-top: 4px;
}

.rule-form :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.dialog-section + .dialog-section {
  margin-top: 20px;
}

.dialog-section-title {
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.dialog-grid-basic {
  grid-template-columns: minmax(0, 1.5fr) minmax(120px, 0.5fr) minmax(140px, 0.7fr);
}

.dialog-grid-secondary {
  margin-top: 4px;
}

.dialog-item {
  min-width: 0;
}

.dialog-item-wide {
  min-width: 0;
}

.match-mode-group {
  display: flex;
  flex-wrap: wrap;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.help {
  margin-top: 6px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.template-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.template-row :deep(.el-input-number) {
  width: 220px;
}

.row-hint {
  color: var(--text-muted);
  font-size: 12px;
}

.label-tip {
  margin-left: 6px;
  cursor: help;
  color: var(--text-muted);
}

.kv-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.kv-row + .kv-row {
  margin-top: 8px;
}

.kv-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.textarea-block :deep(.el-textarea__inner) {
  min-height: 108px;
}

@media (max-width: 900px) {
  .filter-grid {
    grid-template-columns: 1fr 1fr;
  }

  .filter-actions {
    grid-column: 1 / -1;
  }

  .dialog-grid,
  .dialog-grid-basic {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .filter-grid {
    grid-template-columns: 1fr;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .row-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .template-row {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
