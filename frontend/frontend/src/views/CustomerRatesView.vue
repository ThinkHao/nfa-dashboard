<template>
  <div class="rates-view">
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span>客户业务费率筛选</span>
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
        <el-form-item label="学校">
          <el-input v-model="query.school_name" clearable placeholder="学校名称" style="width: 220px" />
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
        <el-table-column prop="school_name" label="学校" min-width="160" show-overflow-tooltip />
        <el-table-column prop="customer_fee" label="客户费" width="120" />
        <el-table-column prop="network_line_fee" label="专线费" width="120" />
        <el-table-column prop="general_fee" label="通用费" width="120" />
        <el-table-column prop="customer_fee_owner_id" label="客户费归属" width="120" />
        <el-table-column prop="network_line_fee_owner_id" label="专线费归属" width="120" />
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

    <el-dialog v-model="dialogVisible" title="新增/更新 客户业务费率" width="560px">
      <el-form :model="form" label-width="140px">
        <el-form-item label="区域" required>
          <el-input v-model="form.region" />
        </el-form-item>
        <el-form-item label="运营商" required>
          <el-input v-model="form.cp" />
        </el-form-item>
        <el-form-item label="学校">
          <el-input v-model="form.school_name" />
        </el-form-item>
        <el-form-item label="客户费">
          <el-input-number v-model="form.customer_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="专线费">
          <el-input-number v-model="form.network_line_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="通用费">
          <el-input-number v-model="form.general_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="客户费归属用户ID">
          <el-input-number v-model="form.customer_fee_owner_id" :min="1" />
        </el-form-item>
        <el-form-item label="专线费归属用户ID">
          <el-input-number v-model="form.network_line_fee_owner_id" :min="1" />
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
import type { RateCustomer, PaginatedData, UpsertRateCustomerRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.customer.write'))

const loading = ref(false)
const items = ref<RateCustomer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ region?: string; cp?: string; school_name?: string }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.school_name) p.school_name = query.school_name
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<RateCustomer> = await api.settlementRates.customer.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
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

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<UpsertRateCustomerRequest>({ region: '', cp: '' })

function openDialog() {
  Object.assign(form, { region: '', cp: '', school_name: undefined, customer_fee: undefined, network_line_fee: undefined, general_fee: undefined, customer_fee_owner_id: undefined, network_line_fee_owner_id: undefined })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp) { ElMessage.warning('区域与运营商为必填'); return }
  saving.value = true
  try {
    await api.settlementRates.customer.upsert(form)
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
