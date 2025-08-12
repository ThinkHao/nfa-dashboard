<template>
  <div class="sys-perms-view">
    <el-card class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>权限设置<span v-if="!canManage">（只读）</span></span>
          <div class="actions" v-if="canManage">
            <el-button type="primary" size="small" @click="openCreateDialog">新建权限</el-button>
            <el-button type="warning" size="small" @click="onSync">从代码同步</el-button>
          </div>
        </div>
      </template>
      <div class="filters">
        <el-input
          v-model="keyword"
          placeholder="搜索：权限名/代码/描述"
          clearable
          style="width: 360px; margin-right: 8px"
          @keyup.enter="onSearch"
          @clear="onSearch"
        />
        <el-button type="primary" @click="onSearch">查询</el-button>
        <el-button @click="onReset">重置</el-button>
      </div>
    </el-card>

    <el-card class="box-card" shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span>权限列表</span>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="分组" width="140">
          <template #default="{ row }">
            <el-tag type="info" effect="plain">{{ groupOf(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="代码" min-width="260" />
        <el-table-column prop="name" label="权限名" min-width="260" />
        <el-table-column prop="description" label="描述" min-width="260" show-overflow-tooltip />
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="200" v-if="canManage">
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

    <!-- 新建权限 -->
    <el-dialog v-model="createVisible" title="新建权限" width="520px">
      <el-form label-width="100px">
        <el-form-item label="代码">
          <el-input v-model="createForm.code" placeholder="domain.resource.action" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createVisible=false">取消</el-button>
          <el-button type="primary" @click="onCreate">保存</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 编辑权限：仅可改名称/描述，代码不可改 -->
    <el-dialog v-model="editVisible" title="编辑权限" width="520px">
      <el-form label-width="100px">
        <el-form-item label="代码">
          <el-input v-model="editForm.code" disabled />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editVisible=false">取消</el-button>
          <el-button type="primary" @click="onUpdate">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, reactive, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { PermissionLite } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const loading = ref(false)
const items = ref<PermissionLite[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('system.permission.manage'))

const createVisible = ref(false)
const editVisible = ref(false)
const createForm = reactive({ code: '', name: '', description: '' as string | null })
const editForm = reactive({ id: 0 as number, code: '', name: '', description: '' as string | null })

function buildParams() { return { page: page.value, page_size: pageSize.value, keyword: keyword.value || undefined } }

async function fetchData() {
  loading.value = true
  try {
    const res = await api.system.permissions.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }
function onSearch() { page.value = 1; fetchData() }
function onReset() { keyword.value = ''; page.value = 1; fetchData() }
function groupOf(p: PermissionLite) {
  const code = p.code || p.name || ''
  if (!code) return '-'
  const idx = code.indexOf('.')
  return idx > 0 ? code.slice(0, idx) : code
}

function openCreateDialog() {
  createForm.code = ''
  createForm.name = ''
  createForm.description = ''
  createVisible.value = true
}

async function onCreate() {
  if (!canManage.value) return
  if (!createForm.code || !createForm.name) {
    return ElMessage.warning('请填写代码和名称')
  }
  try {
    await api.system.permissions.create({ code: createForm.code.trim(), name: createForm.name.trim(), description: createForm.description || undefined })
    ElMessage.success('创建成功')
    createVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '创建失败')
  }
}

function openEditDialog(row: PermissionLite) {
  editForm.id = row.id || 0
  editForm.code = row.code || row.name || ''
  editForm.name = row.name
  editForm.description = (row.description as any) || ''
  editVisible.value = true
}

async function onUpdate() {
  if (!canManage.value) return
  if (!editForm.id) return
  if (!editForm.name) {
    return ElMessage.warning('名称不能为空')
  }
  try {
    await api.system.permissions.update(editForm.id, { name: editForm.name.trim(), description: editForm.description || undefined })
    ElMessage.success('更新成功')
    editVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '更新失败')
  }
}

async function onRemove(row: PermissionLite) {
  if (!canManage.value) return
  try {
    await ElMessageBox.confirm(`确定删除/禁用该权限「${row.name}」？`, '提示', { type: 'warning' })
  } catch { return }
  try {
    if (!row.id) throw new Error('无效的ID')
    await api.system.permissions.remove(row.id)
    ElMessage.success('已删除/禁用')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

async function onSync() {
  if (!canManage.value) return
  try {
    await ElMessageBox.confirm('将从代码同步权限字典（幂等 Upsert），确认继续？', '从代码同步', { type: 'warning' })
  } catch { return }
  try {
    await api.system.permissions.sync()
    ElMessage.success('同步成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '同步失败')
  }
}

onMounted(fetchData)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.pagination { margin-top: 12px; display: flex; justify-content: flex-end; }
.filters { display: flex; align-items: center; gap: 8px; }
.actions { display: flex; align-items: center; gap: 8px; }
</style>
