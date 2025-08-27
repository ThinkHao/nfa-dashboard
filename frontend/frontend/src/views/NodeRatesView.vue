<template>
  <div class="rates-view">
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span>节点业务费率筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" type="success" @click="openDialog()">新增/更新</el-button>
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
        <el-form-item label="结算类型">
          <el-input v-model="query.settlement_type" clearable placeholder="如 IDC" style="width: 160px" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header"><span>费率列表</span></div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column prop="settlement_type" label="结算类型" width="120" />
        <el-table-column prop="cp_fee" label="CP费" width="120" />
        <el-table-column prop="cp_fee_owner_id" label="CP费归属" width="120" />
        <el-table-column prop="node_construction_fee" label="节点建设费" width="120" />
        <el-table-column prop="node_construction_fee_owner_id" label="建设费归属" width="120" />
        <el-table-column prop="rack_fee" label="机柜费" width="120" />
        <el-table-column prop="rack_fee_owner_id" label="机柜费归属" width="120" />
        <el-table-column prop="other_fee" label="其他费" width="120" />
        <el-table-column prop="other_fee_owner_id" label="其他费归属" width="120" />
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

    <el-dialog v-model="dialogVisible" title="新增/更新 节点业务费率" width="580px">
      <el-form :model="form" label-width="160px">
        <el-form-item label="区域" required>
          <el-input v-model="form.region" />
        </el-form-item>
        <el-form-item label="运营商" required>
          <el-input v-model="form.cp" />
        </el-form-item>
        <el-form-item label="结算类型" required>
          <el-input v-model="form.settlement_type" />
        </el-form-item>
        <el-form-item label="CP费">
          <el-input-number v-model="form.cp_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="CP费归属用户ID">
          <el-input-number v-model="form.cp_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="节点建设费">
          <el-input-number v-model="form.node_construction_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="建设费归属用户ID">
          <el-input-number v-model="form.node_construction_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="机柜费">
          <el-input-number v-model="form.rack_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="机柜费归属用户ID">
          <el-input-number v-model="form.rack_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="其他费">
          <el-input-number v-model="form.other_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="其他费归属用户ID">
          <el-input-number v-model="form.other_fee_owner_id" :min="1" />
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
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { RateNode, PaginatedData, UpsertRateNodeRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.node.write'))

const loading = ref(false)
const items = ref<RateNode[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ region?: string; cp?: string; settlement_type?: string }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.settlement_type) p.settlement_type = query.settlement_type
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<RateNode> = await api.settlementRates.node.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, settlement_type: undefined }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<UpsertRateNodeRequest>({ region: '', cp: '', settlement_type: '' })

function openDialog() {
  Object.assign(form, { region: '', cp: '', settlement_type: '', cp_fee: undefined, cp_fee_owner_id: undefined, node_construction_fee: undefined, node_construction_fee_owner_id: undefined, rack_fee: undefined, rack_fee_owner_id: undefined, other_fee: undefined, other_fee_owner_id: undefined })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp || !form.settlement_type) { ElMessage.warning('区域/运营商/结算类型为必填'); return }
  saving.value = true
  try {
    await api.settlementRates.node.upsert(form)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.rates-view { padding: 20px; }
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: 8px; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
