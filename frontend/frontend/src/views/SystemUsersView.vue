<template>
  <div class="sys-users-view">
    <h1 class="page-title">用户管理</h1>
    <el-card class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">用户管理</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button type="primary" plain @click="openCreateUser">新建用户</el-button>
          </div>
        </div>
      </template>
      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="用户名">
          <el-input v-model="query.username" placeholder="支持包含匹配" clearable style="width: 220px" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width: 140px">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="box-card" shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span class="card-title">用户列表</span>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="180" />
        <el-table-column prop="alias" label="别名" width="180" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="角色" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ (row.roles || []).map(r => r.name).join(', ') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            <span>{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEditAlias(row)">编辑别名</el-button>
            <el-button size="small" type="primary" @click="openAssignRoles(row)">分配角色</el-button>
            <el-button
              size="small"
              :type="row.status === 1 ? 'warning' : 'success'"
              :loading="opLoadingId === row.id && opType === 'status'"
              @click="toggleStatus(row)"
            >{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
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
    <!-- 新建用户弹窗 -->
    <el-dialog v-model="createDialogVisible" title="新建用户" width="520px">
      <el-form label-width="90px">
        <el-form-item label="用户名" required>
          <el-input v-model="createForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="别名">
          <el-input v-model="createForm.alias" placeholder="可选，用于显示" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="createForm.password" type="password" show-password placeholder="至少6位" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="可选" />
        </el-form-item>
        <el-form-item label="手机">
          <el-input v-model="createForm.phone" placeholder="可选" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="createForm.status" style="width: 180px">
            <el-option :value="1" label="启用" />
            <el-option :value="0" label="禁用" />
          </el-select>
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role_ids" multiple filterable placeholder="选择角色" style="width: 100%" @visible-change="(v:boolean)=>{ if(v) ensureAllRolesLoaded() }">
            <el-option v-for="r in allRoles" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="createSubmitting" @click="submitCreateUser">确 定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 分配角色弹窗 -->
    <el-dialog v-model="roleDialogVisible" title="分配角色" width="520px">
      <div>
        <div style="margin-bottom: 8px;">用户：{{ roleDialogUser?.username }}</div>
        <el-select v-model="selectedRoleIds" multiple filterable placeholder="选择角色" style="width: 100%">
          <el-option v-for="r in allRoles" :key="r.id" :label="r.name" :value="r.id" />
        </el-select>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="roleDialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="opType === 'roles'" @click="submitAssignRoles">确 定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 编辑别名弹窗 -->
    <el-dialog v-model="editAliasDialogVisible" title="编辑别名" width="420px">
      <div style="margin-bottom: 8px;">用户：{{ editAliasUser?.username }}</div>
      <el-input v-model="editAliasValue" placeholder="留空可清除别名" maxlength="64" show-word-limit />
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editAliasDialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="editAliasSubmitting" @click="submitEditAlias">确 定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { SystemUser, Role } from '@/types/api'

const loading = ref(false)
const items = ref<SystemUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ username?: string; status?: number }>({})

// 操作状态
const opLoadingId = ref<number | null>(null)
const opType = ref<'status' | 'roles' | null>(null)

// 分配角色弹窗
const roleDialogVisible = ref(false)
const roleDialogUser = ref<SystemUser | null>(null)
const allRoles = ref<Role[]>([])
const selectedRoleIds = ref<number[]>([])

// 新建用户弹窗
const createDialogVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive<{ username: string; alias?: string; password: string; email?: string; phone?: string; status: number; role_ids: number[] }>({
  username: '', alias: '', password: '', email: '', phone: '', status: 1, role_ids: []
})

// 编辑别名弹窗
const editAliasDialogVisible = ref(false)
const editAliasSubmitting = ref(false)
const editAliasUser = ref<SystemUser | null>(null)
const editAliasValue = ref<string>('')

function formatTime(ts?: string) {
  if (!ts) return ''
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.username) p.username = query.username
  if (query.status !== undefined) p.status = query.status
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res = await api.system.users.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function openCreateUser() {
  await ensureAllRolesLoaded()
  // reset form
  createForm.username = ''
  createForm.alias = ''
  createForm.password = ''
  createForm.email = ''
  createForm.phone = ''
  createForm.status = 1
  createForm.role_ids = []
  createDialogVisible.value = true
}

async function submitCreateUser() {
  if (!createForm.username?.trim()) { ElMessage.error('请输入用户名'); return }
  if (!createForm.password || createForm.password.length < 6) { ElMessage.error('密码至少6位'); return }
  createSubmitting.value = true
  try {
    const payload: any = {
      username: createForm.username.trim(),
      alias: createForm.alias?.trim() || undefined,
      password: createForm.password,
      status: createForm.status,
    }
    if (createForm.email && createForm.email.trim()) payload.email = createForm.email.trim()
    if (createForm.phone && createForm.phone.trim()) payload.phone = createForm.phone.trim()
    if (createForm.role_ids && createForm.role_ids.length) payload.role_ids = createForm.role_ids
    await api.system.users.create(payload)
    ElMessage.success('创建成功')
    createDialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '创建失败')
  } finally {
    createSubmitting.value = false
  }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { username: undefined, status: undefined }); page.value = 1; pageSize.value = 10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

onMounted(fetchData)

async function toggleStatus(row: SystemUser) {
  const next = row.status === 1 ? 0 : 1
  try {
    await ElMessageBox.confirm(`确定要${next === 1 ? '启用' : '禁用'}用户「${row.username}」吗？`, '确认操作', { type: 'warning' })
  } catch { return }
  opLoadingId.value = row.id; opType.value = 'status'
  try {
    await api.system.users.updateStatus(row.id, { status: next })
    ElMessage.success('操作成功')
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '操作失败')
  } finally {
    opLoadingId.value = null; opType.value = null
  }
}

async function ensureAllRolesLoaded() {
  if (allRoles.value.length > 0) return
  try {
    const res = await api.system.roles.list({ page: 1, page_size: 1000 })
    allRoles.value = res.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载角色列表失败')
  }
}

async function openAssignRoles(row: SystemUser) {
  await ensureAllRolesLoaded()
  roleDialogUser.value = row
  selectedRoleIds.value = (row.roles || []).map(r => r.id)
  roleDialogVisible.value = true
}

async function submitAssignRoles() {
  if (!roleDialogUser.value) return
  opLoadingId.value = roleDialogUser.value.id; opType.value = 'roles'
  try {
    await api.system.users.setRoles(roleDialogUser.value.id, { role_ids: selectedRoleIds.value })
    ElMessage.success('分配成功')
    roleDialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '分配失败')
  } finally {
    opLoadingId.value = null; opType.value = null
  }
}

function openEditAlias(row: SystemUser) {
  editAliasUser.value = row
  editAliasValue.value = row.alias || ''
  editAliasDialogVisible.value = true
}

async function submitEditAlias() {
  if (!editAliasUser.value) return
  const raw = editAliasValue.value || ''
  const trimmed = raw.trim()
  if (trimmed.length > 64) { ElMessage.error('别名长度不能超过64个字符'); return }
  editAliasSubmitting.value = true
  try {
    await api.system.users.updateAlias(editAliasUser.value.id, { alias: trimmed || null })
    ElMessage.success('更新成功')
    editAliasDialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '更新失败')
  } finally {
    editAliasSubmitting.value = false
  }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { margin-top: 12px; display: flex; justify-content: flex-end; }
</style>
