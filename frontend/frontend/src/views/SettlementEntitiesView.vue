<template>
  <div class="entities-view">
    <h1 class="page-title">业务对象</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">业务对象筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" type="success" @click="openCreateDialog">新增</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="类型">
          <el-select v-model="query.entity_type" clearable filterable placeholder="选择类型" style="width: 220px">
            <el-option v-for="bt in btOptions" :key="bt.code" :label="`${bt.name} (${bt.code})`" :value="bt.code" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="query.entity_name" clearable placeholder="对象名称" style="width: 240px" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header"><span class="card-title">业务对象列表</span></div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="entity_type" label="类型" width="160" />
        <el-table-column prop="entity_name" label="名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="contact_info" label="联系信息" min-width="220" show-overflow-tooltip />
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

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑业务对象' : '新增业务对象'" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="类型" required>
          <el-select v-model="form.entity_type" filterable placeholder="选择类型">
            <el-option v-for="bt in btOptions" :key="bt.code" :label="`${bt.name} (${bt.code})`" :value="bt.code" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.entity_name" />
        </el-form-item>
        <el-form-item label="联系信息">
          <el-input v-model="form.contact_info" type="textarea" :rows="3" />
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
import type { BusinessEntity, PaginatedData, CreateBusinessEntityRequest, UpdateBusinessEntityRequest, BusinessType } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('entities.write'))

const loading = ref(false)
const items = ref<BusinessEntity[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ entity_type?: string; entity_name?: string }>({})
const btOptions = ref<BusinessType[]>([])

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.entity_type) p.entity_type = query.entity_type
  if (query.entity_name) p.entity_name = query.entity_name
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<BusinessEntity> = await api.settlementEntities.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadBusinessTypes() {
  try {
    btOptions.value = await api.settlementBusinessTypes.listAllEnabled()
  } catch {}
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { entity_type: undefined, entity_name: undefined }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

// Dialog & CRUD
const dialogVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const form = reactive<{ entity_type: string; entity_name: string; contact_info?: string | null }>({ entity_type: '', entity_name: '', contact_info: '' })

function openCreateDialog() {
  isEdit.value = false
  editingId.value = null
  Object.assign(form, { entity_type: '', entity_name: '', contact_info: '' })
  dialogVisible.value = true
}

function openEditDialog(row: BusinessEntity) {
  isEdit.value = true
  editingId.value = row.id
  Object.assign(form, { entity_type: row.entity_type, entity_name: row.entity_name, contact_info: row.contact_info || '' })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.entity_type || !form.entity_name) { ElMessage.warning('类型与名称为必填'); return }
  saving.value = true
  try {
    if (isEdit.value && editingId.value) {
      const payload: UpdateBusinessEntityRequest = {
        entity_type: form.entity_type,
        entity_name: form.entity_name,
        contact_info: form.contact_info || null,
      }
      await api.settlementEntities.update(editingId.value, payload)
      ElMessage.success('更新成功')
    } else {
      const payload: CreateBusinessEntityRequest = {
        entity_type: form.entity_type,
        entity_name: form.entity_name,
        contact_info: form.contact_info || null,
      }
      await api.settlementEntities.create(payload)
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

async function onRemove(row: BusinessEntity) {
  try {
    await ElMessageBox.confirm(`确认删除业务对象「${row.entity_name}」？`, '提示', { type: 'warning' })
  } catch { return }
  try {
    await api.settlementEntities.remove(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

onMounted(() => { loadBusinessTypes(); fetchData() })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
