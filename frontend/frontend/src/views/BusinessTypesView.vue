<template>
  <div class="bt-view">
    <h1 class="page-title">业务类型管理</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">业务类型筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" type="success" @click="openCreateDialog">新增</el-button>
          </div>
        </div>
      </template>
      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="编码">
          <el-input v-model="query.code" clearable placeholder="如 customer" style="width: 200px" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="query.name" clearable placeholder="名称" style="width: 200px" />
        </el-form-item>
        <el-form-item label="启用">
          <el-select v-model="query.enabled" clearable style="width: 140px" placeholder="全部">
            <el-option :value="true" label="启用" />
            <el-option :value="false" label="禁用" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header"><span class="card-title">业务类型列表</span></div>
      </template>
      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="code" label="编码" width="160" />
        <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column label="启用" width="120">
          <template #default="{ row }">
            <el-tag v-if="!canWrite" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
            <el-switch v-else v-model="row.enabled" @change="onToggleEnabled(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column v-if="canWrite" label="操作" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="onRemove(row)">删除</el-button>
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

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑业务类型' : '新增业务类型'" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="编码" required>
          <el-input v-model="form.code" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
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
import type { BusinessType, PaginatedData, CreateBusinessTypeRequest, UpdateBusinessTypeRequest } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('business_types.write'))

const loading = ref(false)
const items = ref<BusinessType[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ code?: string; name?: string; enabled?: boolean | null }>({ enabled: null })

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.code) p.code = query.code
  if (query.name) p.name = query.name
  if (query.enabled !== null && query.enabled !== undefined) p.enabled = query.enabled
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<BusinessType> = await api.settlementBusinessTypes.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { code: undefined, name: undefined, enabled: null }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

// Dialog & CRUD
const dialogVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const form = reactive<{ code: string; name: string; description?: string | null; enabled: boolean }>({ code: '', name: '', description: '', enabled: true })

function openCreateDialog() {
  isEdit.value = false
  editingId.value = null
  Object.assign(form, { code: '', name: '', description: '', enabled: true })
  dialogVisible.value = true
}

function openEditDialog(row: BusinessType) {
  isEdit.value = true
  editingId.value = row.id
  Object.assign(form, { code: row.code, name: row.name, description: row.description || '', enabled: !!row.enabled })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.code || !form.name) { ElMessage.warning('编码与名称为必填'); return }
  saving.value = true
  try {
    if (isEdit.value && editingId.value) {
      const payload: UpdateBusinessTypeRequest = { name: form.name, description: form.description || null, enabled: form.enabled }
      await api.settlementBusinessTypes.update(editingId.value, payload)
      ElMessage.success('更新成功')
    } else {
      const payload: CreateBusinessTypeRequest = { code: form.code, name: form.name, description: form.description || null, enabled: form.enabled }
      await api.settlementBusinessTypes.create(payload)
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

async function onRemove(row: BusinessType) {
  try { await ElMessageBox.confirm(`确认删除业务类型「${row.name}」？`, '提示', { type: 'warning' }) } catch { return }
  try {
    await api.settlementBusinessTypes.remove(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

async function onToggleEnabled(row: BusinessType) {
  if (!canWrite.value) return
  try {
    await api.settlementBusinessTypes.update(row.id, { enabled: !!row.enabled })
    ElMessage.success('已更新启用状态')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '更新失败')
  }
}

onMounted(fetchData)
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
