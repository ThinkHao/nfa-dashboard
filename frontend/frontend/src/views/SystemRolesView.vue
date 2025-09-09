<template>
  <div class="sys-roles-view">
    <h1 class="page-title">角色管理</h1>
    <el-card class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">角色管理</span>
        </div>
      </template>
    </el-card>

    <el-card class="box-card" shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span class="card-title">角色列表</span>
          <div>
            <el-button type="primary" @click="openCreate">新建角色</el-button>
          </div>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading" @expand-change="onExpandChange">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div style="padding: 8px 16px;">
              <el-skeleton v-if="rolePermLoading[row.id]" :rows="4" animated />
              <el-empty v-else-if="!rolePermMap[row.id] || rolePermMap[row.id].length === 0" description="暂无权限" />
              <div v-else class="perm-tags">
                <el-tag v-for="p in rolePermMap[row.id]" :key="p.id" effect="plain" class="perm-tag">{{ p.name }}</el-tag>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="角色名" width="220" />
        <el-table-column prop="description" label="描述" min-width="260" show-overflow-tooltip />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            <span>{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="380" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openPermPreview(row)">查看权限</el-button>
            <el-button size="small" type="primary" @click="openSetPerms(row)">设置权限</el-button>
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeRole(row)">删除</el-button>
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

    <!-- 新建角色弹窗 -->
    <el-dialog v-model="createVisible" title="新建角色" width="520px">
      <el-form label-width="90px">
        <el-form-item label="角色名" required>
          <el-input v-model="form.name" placeholder="请输入角色名" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createVisible = false">取 消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="submitCreate">确 定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 编辑角色弹窗 -->
    <el-dialog v-model="editVisible" title="编辑角色" width="520px">
      <el-form label-width="90px">
        <el-form-item label="角色名" required>
          <el-input v-model="form.name" placeholder="请输入角色名" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editVisible = false">取 消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="submitEdit">保 存</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 设置权限弹窗 -->
    <el-dialog v-model="permVisible" title="设置权限" width="560px">
      <div style="margin-bottom: 8px;">角色：{{ currentRole?.name }}</div>
      <el-select v-model="selectedPermIds" multiple filterable placeholder="选择权限" style="width: 100%" @visible-change="onPermSelectVisible">
        <el-option v-for="p in allPerms" :key="p.id" :label="p.name" :value="p.id" />
      </el-select>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="permVisible = false">取 消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="submitPerms">确 定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 权限预览抽屉 -->
    <el-drawer v-model="permPreviewVisible" title="角色权限" direction="rtl" size="40%">
      <div style="margin-bottom: 8px;">角色：{{ previewRole?.name }}</div>
      <el-skeleton v-if="previewLoading" :rows="6" animated />
      <el-empty v-else-if="previewPerms.length === 0" description="暂无权限" />
      <el-scrollbar v-else height="60vh">
        <div class="perm-tags">
          <el-tag v-for="p in previewPerms" :key="p.id" effect="plain" class="perm-tag">{{ p.name }}</el-tag>
        </div>
      </el-scrollbar>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { Role, PermissionLite } from '@/types/api'

const loading = ref(false)
const items = ref<Role[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

// dialogs & forms
const submitLoading = ref(false)
const createVisible = ref(false)
const editVisible = ref(false)
const permVisible = ref(false)
const form = reactive<{ id?: number|null; name: string; description?: string }>({ id: null, name: '', description: '' })
const currentRole = ref<Role | null>(null)
const allPerms = ref<PermissionLite[]>([])
const selectedPermIds = ref<number[]>([])
// inline expand preview states
const rolePermMap = reactive<Record<number, PermissionLite[]>>({})
const rolePermLoading = reactive<Record<number, boolean>>({})
// preview drawer
const permPreviewVisible = ref(false)
const previewRole = ref<Role | null>(null)
const previewPerms = ref<PermissionLite[]>([])
const previewLoading = ref(false)

function formatTime(ts?: string) {
  if (!ts) return ''
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function buildParams() { return { page: page.value, page_size: pageSize.value } }

async function fetchData() {
  loading.value = true
  try {
    const res = await api.system.roles.list(buildParams())
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

onMounted(fetchData)

function openCreate() {
  form.id = null; form.name = ''; form.description = ''
  createVisible.value = true
}

async function submitCreate() {
  if (!form.name || !form.name.trim()) { ElMessage.error('请输入角色名'); return }
  submitLoading.value = true
  try {
    await api.system.roles.create({ name: form.name.trim(), description: form.description?.trim() || undefined })
    ElMessage.success('创建成功')
    createVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '创建失败')
  } finally { submitLoading.value = false }
}

function openEdit(row: Role) {
  form.id = row.id; form.name = row.name; form.description = row.description || ''
  editVisible.value = true
}

async function submitEdit() {
  if (!form.id) return
  if (!form.name || !form.name.trim()) { ElMessage.error('请输入角色名'); return }
  submitLoading.value = true
  try {
    await api.system.roles.update(form.id, { name: form.name.trim(), description: form.description?.trim() || undefined })
    ElMessage.success('保存成功')
    editVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally { submitLoading.value = false }
}

async function removeRole(row: Role) {
  try {
    await ElMessageBox.confirm(`确定删除角色「${row.name}」吗？`, '删除确认', { type: 'warning' })
  } catch { return }
  submitLoading.value = true
  try {
    await api.system.roles.remove(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  } finally { submitLoading.value = false }
}

async function ensureAllPermsLoaded() {
  if (allPerms.value.length > 0) return
  try {
    const res = await api.system.permissions.list({ page: 1, page_size: 1000 })
    allPerms.value = res.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载权限列表失败')
  }
}

async function openSetPerms(row: Role) {
  await ensureAllPermsLoaded()
  currentRole.value = row
  try {
    const cur = await api.system.roles.getPermissions(row.id)
    selectedPermIds.value = (cur || []).map(p => p.id!).filter(Boolean) as number[]
  } catch { selectedPermIds.value = [] }
  permVisible.value = true
}

async function submitPerms() {
  if (!currentRole.value) return
  submitLoading.value = true
  try {
    await api.system.roles.setPermissions(currentRole.value.id, { permission_ids: selectedPermIds.value })
    ElMessage.success('设置成功')
    permVisible.value = false
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '设置失败')
  } finally { submitLoading.value = false }
}

function onPermSelectVisible(v: boolean) {
  if (v) ensureAllPermsLoaded()
}

async function onExpandChange(row: Role) {
  const id = row.id
  if (rolePermMap[id] && rolePermMap[id].length > 0) return
  rolePermLoading[id] = true
  try {
    rolePermMap[id] = await api.system.roles.getPermissions(id)
  } catch (e: any) {
    rolePermMap[id] = []
    ElMessage.error(e?.response?.data?.message || e?.message || '加载权限失败')
  } finally {
    rolePermLoading[id] = false
  }
}

async function openPermPreview(row: Role) {
  previewRole.value = row
  permPreviewVisible.value = true
  previewLoading.value = true
  try {
    previewPerms.value = await api.system.roles.getPermissions(row.id)
  } catch (e: any) {
    previewPerms.value = []
    ElMessage.error(e?.response?.data?.message || e?.message || '加载权限失败')
  } finally { previewLoading.value = false }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.pagination { margin-top: 12px; display: flex; justify-content: flex-end; }
.perm-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.perm-tag { margin-bottom: 6px; }
</style>
