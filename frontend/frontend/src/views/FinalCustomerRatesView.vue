<template>
  <div class="rates-view">
    <h1 class="page-title">最终客户费率</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">最终客户费率筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" type="success" @click="openDialog()">新增/更新</el-button>
            <el-button v-if="canWrite" type="warning" :loading="refreshing" @click="onRefresh">初始化并刷新最终费率</el-button>
            <el-button v-if="canWrite" type="danger" :loading="cleaning" @click="onCleanupInvalid">清理无效数据</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="区域">
          <el-input v-model="query.region" clearable placeholder="如 华东" style="width: 160px" />
        </el-form-item>
        <el-form-item label="运营商">
          <el-input v-model="query.cp" clearable placeholder="如 CMCC" style="width: 160px" />
        </el-form-item>
        <el-form-item label="学校">
          <el-input v-model="query.school_name" clearable placeholder="学校名称" style="width: 220px" />
        </el-form-item>
        <!-- 费率类型筛选暂时隐藏 -->
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header"><span class="card-title">费率列表</span></div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column prop="school_name" label="学校" min-width="160" show-overflow-tooltip />
        <!-- 费率类型列暂时隐藏 -->
        <el-table-column prop="final_fee" label="毛利" width="120" />
        <el-table-column prop="customer_fee" label="客户费" width="120" />
        <el-table-column label="客户费归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.customer_fee_owner_id ?? '-'}`">
              <span>{{ displayOwner(row.customer_fee_owner_id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="network_line_fee" label="专线费" width="120" />
        <el-table-column label="专线费归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.network_line_fee_owner_id ?? '-'}`">
              <span>{{ displayOwner(row.network_line_fee_owner_id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="node_deduction_fee" label="节点扣减" width="120" />
        <el-table-column label="扣减归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.node_deduction_fee_owner_id ?? '-'}`">
              <span>{{ displayOwner(row.node_deduction_fee_owner_id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
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

    <el-dialog v-model="dialogVisible" title="新增/更新 最终客户费率" width="620px">
      <el-form :model="form" label-width="170px">
        <el-form-item label="区域" required>
          <el-input v-model="form.region" />
        </el-form-item>
        <el-form-item label="运营商" required>
          <el-input v-model="form.cp" />
        </el-form-item>
        <el-form-item label="学校" required>
          <el-input v-model="form.school_name" />
        </el-form-item>
        <!-- 费率类型表单项暂时隐藏 -->
        <el-form-item label="毛利">
          <el-input-number v-model="form.final_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="客户费">
          <el-input-number v-model="form.customer_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="客户费归属用户ID">
          <el-input-number v-model="form.customer_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="专线费">
          <el-input-number v-model="form.network_line_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="专线费归属用户ID">
          <el-input-number v-model="form.network_line_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="节点扣减费">
          <el-input-number v-model="form.node_deduction_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="节点扣减费归属用户ID">
          <el-input-number v-model="form.node_deduction_fee_owner_id" :min="1" />
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
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { RateFinalCustomer, PaginatedData, UpsertRateFinalCustomerRequest, BusinessEntity } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.final.write'))

const loading = ref(false)
const refreshing = ref(false)
const cleaning = ref(false)
const items = ref<RateFinalCustomer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ region?: string; cp?: string; school_name?: string }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.school_name) p.school_name = query.school_name
  // fee_type 暂不作为筛选参数
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<RateFinalCustomer> = await api.settlementRates.final.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
    // 加载系统用户映射，优先用于显示 owner 名称
    await loadUsersForItems()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, school_name: undefined }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

// 业务对象映射（id -> name），用于展示归属名称
const entityMap = ref<Record<number, string>>({})

async function fetchEntitiesAll() {
  try {
    const pageSize = 1000
    let page = 1
    const map: Record<number, string> = {}
    while (true) {
      const res = await api.settlementEntities.list({ page, page_size: pageSize })
      const list = (res?.items || []) as BusinessEntity[]
      for (const e of list) {
        if (e && typeof e.id === 'number') {
          map[e.id] = e.entity_name
        }
      }
      const total = Number(res?.total || 0)
      if (page * pageSize >= total || list.length === 0) break
      page += 1
    }
    entityMap.value = map
  } catch (_) {
    // 显示归属名是增强体验，失败不阻塞主流程
  }
}

function getEntityName(id?: number | null): string {
  if (id == null) return '-'
  return entityMap.value[id] || `#${id}`
}

// 系统用户映射（id -> 用户基本信息），用于优先显示“系统用户别名/名称”
const userMap = ref<Record<number, { id: number; alias?: string; display_name?: string; username: string }>>({})

// 批量按 items 中出现的 owner_id 拉取系统用户，填充 userMap
async function loadUsersForItems() {
  const ids = new Set<number>()
  for (const r of items.value) {
    if (r?.customer_fee_owner_id != null) {
      const n = Number(r.customer_fee_owner_id); if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
    if (r?.network_line_fee_owner_id != null) {
      const n = Number(r.network_line_fee_owner_id); if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
    if (r?.node_deduction_fee_owner_id != null) {
      const n = Number(r.node_deduction_fee_owner_id); if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
  }
  if (ids.size === 0) { userMap.value = {}; return }
  try {
    const res: any = await api.system.users.list({ ids: Array.from(ids).join(',') })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const m: Record<number, { id: number; alias?: string; display_name?: string; username: string }> = {}
    for (const u of list) {
      if (u && typeof u.id === 'number') m[u.id] = { id: u.id, alias: u.alias, display_name: u.display_name, username: u.username }
    }
    userMap.value = m
  } catch { userMap.value = {} }
}

// 统一的 owner 显示：优先系统用户别名/名称，其次业务对象名，最后 #ID
function displayOwner(id?: number | null): string {
  if (!id) return '-'
  const key = Number(id)
  const u = userMap.value[key]
  if (u) {
    const alias = (u.alias && String(u.alias).trim()) ? String(u.alias).trim() : ''
    const dn = (u.display_name && String(u.display_name).trim()) ? String(u.display_name).trim() : ''
    const un = (u.username && String(u.username).trim()) ? String(u.username).trim() : ''
    return alias || dn || un || `用户#${key}`
  }
  const name = entityMap.value[key]
  return name || `#${key}`
}

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const DEFAULT_FEE_TYPE = 'auto'
const form = reactive<UpsertRateFinalCustomerRequest>({ region: '', cp: '', school_name: '', fee_type: DEFAULT_FEE_TYPE })

function openDialog() {
  Object.assign(form, { region: '', cp: '', school_name: '', fee_type: DEFAULT_FEE_TYPE, final_fee: undefined, customer_fee: undefined, customer_fee_owner_id: undefined, network_line_fee: undefined, network_line_fee_owner_id: undefined, node_deduction_fee: undefined, node_deduction_fee_owner_id: undefined })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp || !form.school_name) { ElMessage.warning('区域/运营商/学校为必填'); return }
  saving.value = true
  try {
    await api.settlementRates.final.upsert(form)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function onRefresh() {
  refreshing.value = true
  try {
    const initAffected = await api.settlementRates.final.initFromCustomer()
    const refreshAffected = await api.settlementRates.final.refresh({})
    ElMessage.success(`初始化 ${initAffected} 条，刷新 ${refreshAffected} 条`)
    fetchData()
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '初始化/刷新失败'
    ElMessage.error(msg)
  } finally {
    refreshing.value = false
  }
}

async function onCleanupInvalid() {
  try {
    await ElMessageBox.confirm('将删除 fee_type=auto 且任意关键费率字段为空的最终费率记录，是否继续？', '确认清理', { type: 'warning', confirmButtonText: '清理', cancelButtonText: '取消' })
  } catch {
    return
  }
  cleaning.value = true
  try {
    const affected = await api.settlementRates.final.cleanupInvalid()
    ElMessage.success(`已清理 ${affected} 条无效记录`)
    fetchData()
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '清理失败'
    ElMessage.error(msg)
  } finally {
    cleaning.value = false
  }
}

onMounted(() => { fetchEntitiesAll(); fetchData() })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
