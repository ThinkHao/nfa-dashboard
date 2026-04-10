<template>
  <div class="discount-rules-view">
    <div class="page-heading">
      <div>
        <h1 class="page-title">折损规则管理</h1>
        <p class="page-subtitle">按范围和字段维护折损规则，用于后续费率结算的折损计算。</p>
      </div>
    </div>

    <el-card shadow="never" class="box-card filter-panel">
      <template #header>
        <div class="card-header">
          <div>
            <span class="card-title">筛选</span>
            <p class="card-subtitle">按名称、范围和启用状态快速定位折损规则。</p>
          </div>
          <div class="header-actions">
            <el-button v-if="canManage" type="success" @click="openEdit()">新增规则</el-button>
          </div>
        </div>
      </template>

      <el-form :model="query" label-position="top" class="filter-form">
        <div class="filter-grid">
          <el-form-item label="名称" class="filter-item">
            <el-input v-model="query.name" clearable placeholder="规则名" class="field-w-full" />
          </el-form-item>
          <el-form-item label="范围" class="filter-item">
            <el-select v-model="query.scope_type" clearable placeholder="选择范围" class="field-w-full">
              <el-option label="global" value="global" />
              <el-option label="region" value="region" />
              <el-option label="cp" value="cp" />
              <el-option label="school" value="school" />
            </el-select>
          </el-form-item>
          <el-form-item label="启用" class="filter-item filter-item-small">
            <el-select v-model="query.enabled" clearable placeholder="全部" class="field-w-full">
              <el-option label="是" :value="'true'" />
              <el-option label="否" :value="'false'" />
            </el-select>
          </el-form-item>
          <div class="filter-actions">
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
          </div>
        </div>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2 list-panel">
      <template #header>
        <div class="card-header">
          <div>
            <span class="card-title">折损规则列表</span>
            <p class="card-subtitle">范围越具体、优先级越靠前的规则越先参与匹配。</p>
          </div>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="scope_type" label="范围" width="100" />
        <el-table-column prop="scope_key" label="范围键" min-width="140" show-overflow-tooltip />
        <el-table-column label="字段" min-width="160">
          <template #default="{ row }">
            <span>{{ formatFields(row.fields) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="100" />
        <el-table-column label="启用" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canManage" label="操作" fixed="right" width="220">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
              <el-button type="primary" link @click="openItems(row)">条目</el-button>
              <el-button type="warning" link @click="toggleEnabled(row)">{{ row.enabled ? '禁用' : '启用' }}</el-button>
              <el-button type="danger" link @click="onRemove(row)">删除</el-button>
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

    <!-- 规则基础信息编辑弹窗 -->
    <el-dialog v-model="editVisible" :title="editForm.id ? '编辑规则' : '新增规则'" width="720px">
      <el-form :model="editForm" label-position="top" class="rule-form">
        <div class="dialog-section">
          <div class="dialog-section-title">基础信息</div>
          <div class="dialog-grid dialog-grid-basic">
            <el-form-item label="名称" required class="dialog-item dialog-item-wide">
              <el-input v-model="editForm.name" />
            </el-form-item>
            <el-form-item label="优先级" class="dialog-item">
              <el-input-number v-model="editForm.priority" :min="0" :step="1" controls-position="right" class="field-w-full" />
            </el-form-item>
            <el-form-item label="启用" class="dialog-item">
              <el-switch v-model="editForm.enabled" />
            </el-form-item>
          </div>
        </div>

        <div class="dialog-section">
          <div class="dialog-section-title">适用范围</div>
          <div class="dialog-grid">
            <el-form-item label="范围类型" required class="dialog-item">
              <el-select v-model="editForm.scope_type" class="field-w-full">
                <el-option label="global" value="global" />
                <el-option label="region" value="region" />
                <el-option label="cp" value="cp" />
                <el-option label="school" value="school" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="editForm.scope_type === 'region'" label="区域" class="dialog-item">
              <el-select v-model="editForm.scope_key" filterable placeholder="选择区域" class="field-w-full">
                <el-option v-for="r in regionOptions" :key="r" :label="r" :value="r" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="editForm.scope_type === 'cp'" label="CP" class="dialog-item">
              <el-select v-model="editForm.scope_key" filterable placeholder="选择 CP" class="field-w-full">
                <el-option v-for="c in cpOptions" :key="c" :label="c" :value="c" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="editForm.scope_type === 'school'" label="学校" class="dialog-item dialog-item-wide">
              <el-select
                v-model="editForm.scope_key"
                clearable
                filterable
                remote
                :remote-method="remoteSearchSchools"
                :loading="schoolsLoading"
                placeholder="搜索学校"
                class="field-w-full"
              >
                <el-option v-for="s in schoolOptions" :key="s" :label="s" :value="s" />
              </el-select>
            </el-form-item>
          </div>
        </div>

        <div class="dialog-section">
          <div class="dialog-section-title">作用字段</div>
          <el-form-item label="字段范围">
            <el-checkbox-group v-model="fieldsSelected" class="field-group">
              <el-checkbox label="customer_fee">客户费率</el-checkbox>
              <el-checkbox label="network_line_fee">线路费率</el-checkbox>
              <el-checkbox label="general_fee">节点通用费率</el-checkbox>
              <el-checkbox label="channel_rate">渠道费率</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="editVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 规则条目编辑弹窗 -->
    <el-dialog v-model="itemsVisible" title="编辑规则条目" width="960px" :append-to-body="true">
      <div class="items-toolbar">
        <div v-if="!itemsAdvancedJsonMode" class="toolbar-group">
          <el-button @click="onQuickFillStandard">快速填充 100%/75%/25%</el-button>
          <el-button @click="addRow">新增区间</el-button>
        </div>
        <label class="inline-check">
          <el-checkbox v-model="itemsAdvancedJsonMode">JSON 编辑</el-checkbox>
        </label>
      </div>
      <div v-if="!itemsAdvancedJsonMode">
        <el-table :data="itemsRows" border size="small" max-height="480">
          <el-table-column label="开始年(from)" width="160">
            <template #default="{ row }">
              <el-input-number v-model="row.from_year" size="small" :min="1" :step="1" controls-position="right" class="field-w-120" />
            </template>
          </el-table-column>
          <el-table-column label="结束年(to)" width="220">
            <template #default="{ row }">
              <div class="d-flex items-center gap-6">
                <el-input-number v-model="row.to_year" size="small" :min="row.from_year || 1" :step="1" controls-position="right" class="field-w-120" />
                <el-button text type="primary" @click="row.to_year = null">无上限</el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="折损比率(0~1)" width="220">
            <template #default="{ row }">
              <div class="d-flex items-center gap-6">
                <el-input-number v-model="row.discount_rate" size="small" :min="0" :max="1" :step="0.01" :precision="2" controls-position="right" class="field-w-140" />
                <span class="rate-percent">{{ Math.round((Number(row.discount_rate)||0)*100) }}%</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ $index }">
              <el-button text type="danger" @click="removeRow($index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-alert type="info" show-icon class="mt-8"
          title="区间按开始年升序；结束年为空表示无上限；比率范围 0~1，例如 0.75=75%" />
      </div>
      <div v-else>
        <el-alert title='以 JSON 数组形式编辑条目：[ {"from_year":1, "to_year":3, "discount_rate":1 }, {"from_year":2, "to_year":2, "discount_rate":0.75 }, {"from_year":3, "discount_rate":0.25 } ]' type="info" show-icon class="mb-8" />
        <el-input v-model="itemsText" type="textarea" :rows="12" placeholder='[ {"from_year":1, "to_year":3, "discount_rate":0.9} ]' />
      </div>
      <template #footer>
        <el-button @click="itemsVisible=false">取消</el-button>
        <el-button type="primary" :loading="savingItems" @click="onSaveItems">保存条目</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('rates.discount_rule.manage'))

const loading = ref(false)
const items = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ name?: string; scope_type?: string; enabled?: string | '' }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.name) p.name = query.name
  if (query.scope_type) p.scope_type = query.scope_type
  if (query.enabled === 'true' || query.enabled === 'false') p.enabled = query.enabled
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await api.settlementRates.discountRules.list(buildParams())
    items.value = Array.isArray(res?.items) ? res.items : []
    total.value = Number(res?.total || 0)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally { loading.value = false }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { name: undefined, scope_type: undefined, enabled: '' as any }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

// 编辑弹窗
const editVisible = ref(false)
const saving = ref(false)
const editForm = reactive<any>({ id: undefined, name: '', scope_type: 'global', scope_key: '', enabled: true, priority: 100, fields: [] })
const fieldsText = ref('')
const fieldsSelected = ref<string[]>([])

const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])
const schoolOptions = ref<string[]>([])
const schoolsLoading = ref(false)

function openEdit(row?: any) {
  if (row) {
    editForm.id = row.id
    editForm.name = row.name
    editForm.scope_type = row.scope_type
    editForm.scope_key = row.scope_key || ''
    editForm.enabled = !!row.enabled
    editForm.priority = Number(row.priority || 100)
    fieldsText.value = formatFieldsRaw(row.fields)
    const parsed = parseFieldsList(row.fields)
    fieldsSelected.value = parsed.length > 0 ? parsed : ['customer_fee']
  } else {
    editForm.id = undefined
    editForm.name = ''
    editForm.scope_type = 'global'
    editForm.scope_key = ''
    editForm.enabled = true
    editForm.priority = 100
    fieldsText.value = '["customer_fee"]'
    fieldsSelected.value = ['customer_fee']
  }
  editVisible.value = true
}

function formatFieldsRaw(v: any): string {
  try {
    if (!v) return ''
    if (typeof v === 'string') return JSON.stringify(JSON.parse(v))
    return JSON.stringify(v)
  } catch { return String(v) }
}

function formatFields(v: any): string {
  try {
    if (!v) return '-'
    const arr = typeof v === 'string' ? JSON.parse(v) : v
    if (Array.isArray(arr)) return arr.join(',')
    return '-'
  } catch { return '-' }
}

function parseFieldsList(v: any): string[] {
  try {
    if (!v) return []
    const arr = typeof v === 'string' ? JSON.parse(v) : v
    if (Array.isArray(arr)) return arr.filter((x: any) => typeof x === 'string' && x)
    return []
  } catch { return [] }
}

async function onSave() {
  if (!editForm.name) { ElMessage.warning('名称必填'); return }
  if (!editForm.scope_type) { ElMessage.warning('范围必选'); return }
  // 使用点选复选框的结果；为空则默认 customer_fee
  const selected = Array.isArray(fieldsSelected.value) && fieldsSelected.value.length > 0
    ? fieldsSelected.value
    : ['customer_fee']
  const payload: any = {
    name: editForm.name,
    scope_type: editForm.scope_type,
    scope_key: editForm.scope_type === 'global' ? null : (editForm.scope_key || null),
    enabled: !!editForm.enabled,
    priority: Number(editForm.priority || 100),
    fields: selected,
  }
  saving.value = true
  try {
    if (editForm.id) {
      await api.settlementRates.discountRules.update(Number(editForm.id), payload)
    } else {
      await api.settlementRates.discountRules.create({ ...payload, items: [] })
    }
    ElMessage.success('保存成功')
    editVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { saving.value = false }
}

// 条目编辑
const itemsVisible = ref(false)
const savingItems = ref(false)
const itemsForm = reactive<{ id?: number; name?: string }>({})
const itemsText = ref('')
const itemsAdvancedJsonMode = ref(false)
const itemsRows = ref<Array<{ from_year: number; to_year: number | null; discount_rate: number }>>([])

async function openItems(row: any) {
  itemsForm.id = Number(row?.id || 0)
  itemsText.value = ''
  try {
    const data: any = await api.settlementRates.discountRules.get(itemsForm.id!)
    const list = Array.isArray(data?.items) ? data.items : []
    const mapped = list.map((it: any) => ({ from_year: Number(it.from_year), to_year: it.to_year != null ? Number(it.to_year) : null, discount_rate: Number(it.discount_rate) }))
    itemsText.value = JSON.stringify(mapped, null, 2)
    itemsRows.value = mapped
  } catch {}
  itemsVisible.value = true
}

async function onSaveItems() {
  if (!itemsForm.id) return
  let arr: any[] = []
  if (itemsAdvancedJsonMode.value) {
    try {
      const parsed = JSON.parse(itemsText.value || '[]')
      if (!Array.isArray(parsed)) throw new Error('not array')
      arr = parsed.map((x: any) => ({ from_year: Number(x.from_year), to_year: x.to_year != null ? Number(x.to_year) : null, discount_rate: Number(x.discount_rate) }))
    } catch { ElMessage.error('条目 JSON 格式错误'); return }
  } else {
    // 从表格模式构造，并做基础校验
    const rows = (itemsRows.value || []).map(r => ({
      from_year: Number(r.from_year),
      to_year: r.to_year == null ? null : Number(r.to_year),
      discount_rate: Number(r.discount_rate),
    }))
    // 过滤无效行
    const valid = rows.filter(r => r.from_year >= 1 && !Number.isNaN(r.from_year) && !Number.isNaN(r.discount_rate))
    // 排序
    valid.sort((a, b) => a.from_year - b.from_year)
    // 重叠检查：要求后一个区间的 from_year > 前一个的 to_year（若前一个 to_year 为 null，则必须是最后一行）
    let lastEnd: number | null = null
    for (let i = 0; i < valid.length; i++) {
      const r = valid[i]
      if (r.to_year != null && r.to_year < r.from_year) { ElMessage.error(`第 ${i + 1} 行：结束年不能小于开始年`); return }
      if (lastEnd != null && r.from_year <= lastEnd) { ElMessage.error(`第 ${i + 1} 行与上一行区间重叠`); return }
      lastEnd = r.to_year == null ? Number.POSITIVE_INFINITY : r.to_year
      if (r.discount_rate < 0 || r.discount_rate > 1 || Number.isNaN(r.discount_rate)) { ElMessage.error(`第 ${i + 1} 行：折损比率需在 0~1 之间`); return }
      // 若某行无上限（to_year=null），必须是最后一行
      if (r.to_year == null && i !== valid.length - 1) { ElMessage.error(`第 ${i + 1} 行为无上限，应放在最后`); return }
    }
    arr = valid
    // 同步 JSON 文本
    itemsText.value = JSON.stringify(arr, null, 2)
  }
  savingItems.value = true
  try {
    await api.settlementRates.discountRules.replaceItems(itemsForm.id!, arr)
    ElMessage.success('已保存条目')
    itemsVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { savingItems.value = false }
}

function addRow() {
  const last = itemsRows.value[itemsRows.value.length - 1]
  const nextFrom = last ? (last.to_year != null ? Number(last.to_year) + 1 : Number(last.from_year) + 1) : 1
  itemsRows.value.push({ from_year: nextFrom, to_year: nextFrom, discount_rate: 1 })
}

function removeRow(index: number) {
  if (index >= 0 && index < itemsRows.value.length) itemsRows.value.splice(index, 1)
}

function onQuickFillStandard() {
  itemsRows.value = [
    { from_year: 1, to_year: 1, discount_rate: 1 },
    { from_year: 2, to_year: 2, discount_rate: 0.75 },
    { from_year: 3, to_year: null, discount_rate: 0.25 },
  ]
  itemsAdvancedJsonMode.value = false
}

async function toggleEnabled(row: any) {
  try {
    await api.settlementRates.discountRules.update(Number(row.id), { enabled: !row.enabled })
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '操作失败')
  }
}

async function onRemove(row: any) {
  try {
    await ElMessageBox.confirm(`确认删除规则“${row.name}”？`, '确认删除', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
  } catch { return }
  try {
    await api.settlementRates.discountRules.remove(Number(row.id))
    ElMessage.success('已删除')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

async function loadRegionsAndCPs() {
  try {
    const [regions, cps] = await Promise.all([
      (api as any).v2.getRegions(),
      (api as any).v2.getCPs(),
    ])
    regionOptions.value = Array.isArray(regions) ? regions.filter((v: any) => v && v !== 'NULL') : []
    cpOptions.value = Array.isArray(cps) ? cps.filter((v: any) => v && v !== 'NULL') : []
  } catch { regionOptions.value = []; cpOptions.value = [] }
}

async function remoteSearchSchools(q: string) {
  schoolsLoading.value = true
  try {
    const data = await (api as any).v2.getSchools({ school_name: q || undefined, limit: 100, offset: 0 })
    const list: any[] = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
    const names = list
      .map((it: any) => it?.school_name || it?.name || it)
      .map((s: any) => (s != null ? String(s).trim() : ''))
      .filter((s: string) => !!s)
    const uniq = Array.from(new Set(names))
    schoolOptions.value = uniq
  } catch { schoolOptions.value = [] }
  finally { schoolsLoading.value = false }
}
// 支持通过路由参数直接打开编辑/条目
const route = useRoute()

async function handleRouteOpen() {
  try {
    const open = String((route.query as any)?.open || '')
    const id = Number((route.query as any)?.id || 0)
    if (!open || !Number.isFinite(id) || id <= 0) return
    const row = (items.value || []).find((r: any) => Number(r?.id) === id)
    if (open === 'edit') {
      if (row) openEdit(row)
      else {
        try {
          const d: any = await api.settlementRates.discountRules.get(id)
          if (d?.rule) openEdit(d.rule)
        } catch {}
      }
    } else if (open === 'items') {
      openItems(row || { id })
    }
  } catch {}
}

onMounted(async () => {
  try { await Promise.all([fetchData(), loadRegionsAndCPs()]) } finally { handleRouteOpen() }
})

watch(() => route.query, () => { handleRouteOpen() })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }

.filter-panel,
.list-panel {
  overflow: hidden;
}

.page-subtitle {
  margin: 6px 0 0;
  color: var(--text-muted);
  line-height: 1.6;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}

.card-subtitle {
  margin: 6px 0 0;
  color: var(--text-muted);
  line-height: 1.5;
}

.header-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
  flex: 0 0 auto;
}

.filter-form :deep(.el-form-item) {
  margin-bottom: 0;
}

.filter-grid {
  display: grid;
  grid-template-columns: minmax(220px, 1.3fr) minmax(180px, 0.9fr) minmax(160px, 0.7fr) auto;
  gap: 12px 16px;
  align-items: end;
}

.filter-item,
.dialog-item,
.dialog-item-wide {
  min-width: 0;
}

.filter-item-small {
  max-width: 220px;
}

.filter-actions,
.row-actions,
.toolbar-group {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.row-actions {
  align-items: center;
}

.rule-form {
  padding-top: 4px;
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
  grid-template-columns: minmax(0, 1.8fr) minmax(140px, 0.8fr) minmax(120px, 0.6fr);
}

.field-group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
}

.items-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.inline-check {
  display: inline-flex;
  align-items: center;
}

.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.rate-percent { color: var(--text-muted); min-width: 40px; }

@media (max-width: 960px) {
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

@media (max-width: 720px) {
  .filter-grid {
    grid-template-columns: 1fr;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>


