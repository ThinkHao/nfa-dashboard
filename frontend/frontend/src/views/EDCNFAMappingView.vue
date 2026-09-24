<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  ElButton, ElCard, ElDatePicker, ElDialog, ElForm, ElFormItem, ElInput,
  ElMessage, ElMessageBox, ElOption, ElSelect, ElSwitch, ElTable, ElTableColumn,
  ElTag,
} from 'element-plus'
import api from '../api'

interface Entity {
  id: number
  edc_name: string
  display_name: string
  region: string
  cp: string
  entity_type?: string
}

interface Member {
  entity_id: number
  edc_name?: string
  display_name?: string
  valid_from?: string | null
  valid_to?: string | null
  enabled?: boolean
}

interface Mapping {
  id: number
  group_name: string
  nfa_src_region: string
  nfa_cp: string
  enabled: boolean
  remark?: string
  members: Member[]
}

interface MemberRow {
  entity_id: number
  range: [string, string] | null
}

const mappings = ref<Mapping[]>([])
const entities = ref<Entity[]>([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editingID = ref(0)
const form = ref({ group_name: '', nfa_src_region: '', nfa_cp: '', enabled: true, remark: '' })
const memberRows = ref<MemberRow[]>([])

const entityMap = computed(() => new Map(entities.value.map((entity) => [entity.id, entity])))
const selectedEntityIDs = computed(() => memberRows.value.map((row) => row.entity_id))

function entityLabel(id: number): string {
  const entity = entityMap.value.get(id)
  if (!entity) return `实体 ${id}`
  return `${entity.edc_name || entity.display_name}（${entity.region}/${entity.cp}）`
}

function resetForm() {
  editingID.value = 0
  form.value = { group_name: '', nfa_src_region: '', nfa_cp: '', enabled: true, remark: '' }
  memberRows.value = []
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: Mapping) {
  editingID.value = row.id
  form.value = {
    group_name: row.group_name,
    nfa_src_region: row.nfa_src_region,
    nfa_cp: row.nfa_cp,
    enabled: row.enabled,
    remark: row.remark || '',
  }
  memberRows.value = (row.members || []).map((member) => ({
    entity_id: member.entity_id,
    range: member.valid_from && member.valid_to ? [member.valid_from, member.valid_to] : null,
  }))
  dialogVisible.value = true
}

function onMembersChange(ids: number[]) {
  const old = new Map(memberRows.value.map((row) => [row.entity_id, row]))
  memberRows.value = ids.map((id) => old.get(id) || ({ entity_id: id, range: null }))
}

function buildPayload() {
  return {
    ...form.value,
    members: memberRows.value.map((row) => ({
      entity_id: row.entity_id,
      valid_from: row.range?.[0] || null,
      valid_to: row.range?.[1] || null,
    })),
  }
}

async function loadData() {
  loading.value = true
  try {
    const [mappingData, entityData] = await Promise.all([
      (api as any).v2.edcNfa.listMappings(),
      (api as any).v2.edcNfa.listMappingEntities(),
    ])
    mappings.value = Array.isArray(mappingData) ? mappingData : (mappingData?.items || [])
    entities.value = Array.isArray(entityData) ? entityData : (entityData?.items || [])
  } catch (error: any) {
    ElMessage.error(error?.message || '获取映射配置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.value.group_name.trim() || !form.value.nfa_src_region.trim() || !form.value.nfa_cp.trim()) {
    ElMessage.warning('比较组名称、NFA 节点源区域和 CP 均为必填')
    return
  }
  if (memberRows.value.length === 0) {
    ElMessage.warning('至少选择一个 EDC 成员')
    return
  }
  const payload = buildPayload()
  try {
    await ElMessageBox.confirm(
      `将${editingID.value ? '更新' : '创建'}“${form.value.group_name}”，绑定 ${memberRows.value.length} 个 EDC 实体到节点源区域 ${form.value.nfa_src_region} / ${form.value.nfa_cp}。是否继续？`,
      '保存前预览',
      { type: 'warning', confirmButtonText: '确认保存', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  saving.value = true
  try {
    if (editingID.value) await (api as any).v2.edcNfa.updateMapping(editingID.value, payload)
    else await (api as any).v2.edcNfa.createMapping(payload)
    ElMessage.success('映射配置已保存')
    dialogVisible.value = false
    await loadData()
  } catch (error: any) {
    ElMessage.error(error?.message || '保存映射配置失败')
  } finally {
    saving.value = false
  }
}

async function toggle(row: Mapping) {
  try {
    await ElMessageBox.confirm(`确认${row.enabled ? '停用' : '启用'}“${row.group_name}”？`, '状态变更确认', { type: 'warning' })
    await (api as any).v2.edcNfa.setMappingEnabled(row.id, !row.enabled)
    row.enabled = !row.enabled
    ElMessage.success('状态已更新')
  } catch {
    // 用户取消或请求失败时保持当前状态
  }
}

onMounted(loadData)
</script>

<template>
  <div class="mapping-page">
    <ElCard shadow="never">
      <div class="toolbar">
        <div>
          <div class="title">EDC/NFA 映射配置</div>
          <div class="hint">配置页是比较组的唯一维护入口；比较页面只读取已启用映射。</div>
        </div>
        <ElButton type="primary" @click="openCreate">新建比较组</ElButton>
      </div>
    </ElCard>

    <ElCard shadow="never">
      <ElTable v-loading="loading" :data="mappings" stripe empty-text="暂无映射配置">
        <ElTableColumn prop="group_name" label="比较组" min-width="180" />
        <ElTableColumn label="NFA 维度" min-width="200">
          <template #default="scope">{{ scope.row.nfa_src_region }} / {{ scope.row.nfa_cp }}</template>
        </ElTableColumn>
        <ElTableColumn label="EDC 成员" min-width="300">
          <template #default="scope">
            <span>{{ (scope.row.members || []).map((member: Member) => member.edc_name || member.display_name || member.entity_id).join('、') }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="90">
          <template #default="scope"><ElTag :type="scope.row.enabled ? 'success' : 'info'">{{ scope.row.enabled ? '启用' : '停用' }}</ElTag></template>
        </ElTableColumn>
        <ElTableColumn prop="remark" label="备注" min-width="180" show-overflow-tooltip />
        <ElTableColumn label="操作" width="170" fixed="right">
          <template #default="scope">
            <ElButton link type="primary" @click="openEdit(scope.row)">编辑</ElButton>
            <ElButton link :type="scope.row.enabled ? 'danger' : 'success'" @click="toggle(scope.row)">{{ scope.row.enabled ? '停用' : '启用' }}</ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ElDialog v-model="dialogVisible" :title="editingID ? '编辑比较组' : '新建比较组'" width="760px" destroy-on-close>
      <ElForm label-width="100px">
        <ElFormItem label="比较组名称"><ElInput v-model="form.group_name" placeholder="如 SH-jinshan" /></ElFormItem>
        <ElFormItem label="NFA 节点源区域"><ElInput v-model="form.nfa_src_region" placeholder="如 上海市" /></ElFormItem>
        <ElFormItem label="NFA CP"><ElInput v-model="form.nfa_cp" placeholder="如 jinshan" /></ElFormItem>
        <ElFormItem label="EDC 成员">
          <ElSelect
            :model-value="selectedEntityIDs"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            class="full-width"
            placeholder="选择启用中的非备份 EDC 实体"
            @update:model-value="onMembersChange"
          >
            <ElOption v-for="entity in entities" :key="entity.id" :label="entityLabel(entity.id)" :value="entity.id" />
          </ElSelect>
        </ElFormItem>
        <ElTable v-if="memberRows.length" :data="memberRows" size="small" class="member-table">
          <ElTableColumn label="成员" min-width="260">
            <template #default="scope">{{ entityLabel(scope.row.entity_id) }}</template>
          </ElTableColumn>
          <ElTableColumn label="生效区间（可选）" min-width="330">
            <template #default="scope">
              <ElDatePicker v-model="scope.row.range" type="datetimerange" value-format="YYYY-MM-DD HH:mm:ss" range-separator="至" start-placeholder="不限" end-placeholder="不限" />
            </template>
          </ElTableColumn>
        </ElTable>
        <ElFormItem label="启用"><ElSwitch v-model="form.enabled" /></ElFormItem>
        <ElFormItem label="备注"><ElInput v-model="form.remark" type="textarea" :rows="2" /></ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">保存</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
.mapping-page { display: flex; flex-direction: column; gap: 16px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.title { font-size: 18px; font-weight: 600; }
.hint { margin-top: 6px; color: var(--el-text-color-secondary); font-size: 13px; }
.full-width { width: 100%; }
.member-table { margin: -4px 0 16px 100px; width: calc(100% - 100px); }
</style>
