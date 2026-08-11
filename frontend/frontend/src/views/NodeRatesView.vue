<template>
  <div class="rates-view">
    <h1 class="page-title">节点业务费率</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">节点业务费率筛选</span>
          <div>
            <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" type="primary" @click="openGroupDialog()">新建结算分组</el-button>
            <el-button v-if="canWrite" type="success" @click="openDialog()">新增/更新</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="区域">
          <SearchSelect v-model="query.region" :options="regionOptions" clearable placeholder="选择区域" class="field-w-160" />
        </el-form-item>
        <el-form-item label="CP">
          <SearchSelect v-model="query.cp" :options="cpOptions" clearable placeholder="选择 CP" class="field-w-160" />
        </el-form-item>
        <el-form-item label="结算类型">
          <el-select v-model="query.settlement_type" clearable class="field-w-180">
            <el-option label="日95均值" value="daily_95_avg" />
            <el-option label="月95" value="range_95" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header">
          <span class="card-title">结算分组</span>
          <el-button size="small" @click="loadGroups">刷新</el-button>
        </div>
      </template>

      <el-table :data="groups" border stripe v-loading="groupLoading">
        <el-table-column prop="group_name" label="分组名称" min-width="180" />
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column label="成员节点" min-width="260">
          <template #default="{ row }">{{ groupMemberNames(row).join('、') || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="plain">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDialogFromGroup(row)">配置费率</el-button>
            <el-button link type="primary" @click="openGroupDialog(row)">编辑</el-button>
            <el-button v-if="row.enabled" link type="danger" @click="disableGroup(row)">禁用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header"><span class="card-title">费率列表</span></div>
      </template>

      <el-table :data="displayItems" border stripe height="600px" v-loading="loading">
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column prop="display_name" label="节点" min-width="160">
          <template #default="{ row }">
            <el-tag v-if="row.billing_subject_type === 'group'" size="small" type="warning" effect="plain">分组</el-tag>
            <el-tag v-else-if="!row.entity_id" size="small" type="info" effect="plain">默认</el-tag>
            <el-tag v-else size="small" effect="plain">节点</el-tag>
            <span class="rate-subject-name">{{ row.billing_display_name || row.display_name || '区域+CP默认' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="entity_id" label="实体ID" width="90" />
        <el-table-column label="已配置模式" width="170">
          <template #default="{ row }">
            <el-tag v-if="hasMode(row, 'daily_95_avg')" size="small" type="primary" effect="plain">日95均值</el-tag>
            <el-tag v-if="hasMode(row, 'range_95')" size="small" type="success" effect="plain" class="mode-tag">月95</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="日95单价" width="120">
          <template #default="{ row }">{{ modeRateValue(row, 'daily_95_avg', 'node_construction_fee') }}</template>
        </el-table-column>
        <el-table-column label="月95单价" width="120">
          <template #default="{ row }">{{ modeRateValue(row, 'range_95', 'node_construction_fee') }}</template>
        </el-table-column>
        <el-table-column label="日95进制" width="100">
          <template #default="{ row }">{{ modeRateValue(row, 'daily_95_avg', 'unit_base') }}</template>
        </el-table-column>
        <el-table-column label="月95进制" width="100">
          <template #default="{ row }">{{ modeRateValue(row, 'range_95', 'unit_base') }}</template>
        </el-table-column>
        <el-table-column label="归属摘要" min-width="220">
          <template #default="{ row }">
            <el-tooltip placement="top" effect="light" :disabled="ownerSummary(row) === '-'" popper-class="node-rate-owner-tooltip">
              <template #content>
                <div class="owner-tooltip">
                  <div v-for="mode in ownerTooltipModes(row)" :key="mode.mode" class="owner-tooltip-mode">
                    <div class="owner-tooltip-title">{{ mode.label }}</div>
                    <div v-for="item in mode.items" :key="item.label" class="owner-tooltip-row">
                      <span>{{ item.label }}</span>
                      <strong>{{ item.owner }}</strong>
                    </div>
                  </div>
                </div>
              </template>
              <span class="owner-summary">{{ ownerSummary(row) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDialog(row)">编辑</el-button>
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

    <el-drawer :key="dialogKey" v-model="dialogVisible" title="编辑节点业务费率" size="760px" destroy-on-close>
      <el-form :model="form" label-width="150px" class="node-rate-form">
        <div class="drawer-section">
          <div class="section-title">节点范围</div>
          <el-form-item label="区域" required>
            <SearchSelect v-model="form.region" :options="regionOptions" placeholder="选择区域" class="field-w-300" @change="onEntityScopeChange" />
          </el-form-item>
          <el-form-item label="CP" required>
            <SearchSelect v-model="form.cp" :options="cpOptions" placeholder="选择 CP" class="field-w-300" @change="onEntityScopeChange" />
          </el-form-item>
          <el-form-item label="节点">
            <SearchSelect
              v-model="selectedEntityID"
              :disabled="isGroupRateForm"
              clearable
              remote
              :remote-method="remoteSearchEntities"
              :loading="entitiesLoading"
              :options="entityOptions"
              label-key="label"
              value-key="id"
              placeholder="搜索并选择 EDC 节点；留空表示区域+CP默认费率"
              class="field-w-360"
              @change="onEntityChange"
              @clear="onEntityClear"
              @visible-change="onEntityDropdownVisible"
            >
              <template #option="{ option }">
                <div class="entity-option">
                  <span class="entity-option-name">{{ (option as EDCEntityOption).alias || (option as EDCEntityOption).display_name }}</span>
                  <span class="entity-option-meta">{{ (option as EDCEntityOption).region }} / {{ (option as EDCEntityOption).cp }}</span>
                </div>
              </template>
            </SearchSelect>
          </el-form-item>
          <el-form-item label="节点展示名">
            <el-input v-model="form.display_name" :disabled="isGroupRateForm" placeholder="留空表示区域+CP默认费率" />
          </el-form-item>
          <el-form-item v-if="isGroupRateForm" label="计费主体">
            <el-tag type="warning" effect="plain">{{ form.billing_display_name }}</el-tag>
          </el-form-item>
        </div>

        <div class="mode-grid">
          <section class="mode-panel">
            <div class="mode-panel-header">
              <div>
                <div class="mode-title">日95均值</div>
                <div class="mode-subtitle">启用且填写流量单价后，日结算任务才会入库</div>
              </div>
              <el-switch v-model="modeForm.daily_95_avg.enabled" />
            </div>
            <el-form-item label="结算进制">
              <el-select v-model="modeForm.daily_95_avg.unit_base" class="field-w-160" :disabled="!modeForm.daily_95_avg.enabled">
                <el-option label="1000" :value="1000" />
                <el-option label="1024" :value="1024" />
              </el-select>
            </el-form-item>
            <el-form-item label="流量单价（节点建设费）">
              <el-input-number v-model="modeForm.daily_95_avg.node_construction_fee" :disabled="!modeForm.daily_95_avg.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="建设费归属">
              <SearchSelect v-model="modeForm.daily_95_avg.node_construction_fee_owner_id" remote clearable :disabled="!modeForm.daily_95_avg.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="CP费">
              <el-input-number v-model="modeForm.daily_95_avg.cp_fee" :disabled="!modeForm.daily_95_avg.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="CP费归属">
              <SearchSelect v-model="modeForm.daily_95_avg.cp_fee_owner_id" remote clearable :disabled="!modeForm.daily_95_avg.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="机柜费">
              <el-input-number v-model="modeForm.daily_95_avg.rack_fee" :disabled="!modeForm.daily_95_avg.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="机柜费归属">
              <SearchSelect v-model="modeForm.daily_95_avg.rack_fee_owner_id" remote clearable :disabled="!modeForm.daily_95_avg.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="其他费">
              <el-input-number v-model="modeForm.daily_95_avg.other_fee" :disabled="!modeForm.daily_95_avg.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="其他费归属">
              <SearchSelect v-model="modeForm.daily_95_avg.other_fee_owner_id" remote clearable :disabled="!modeForm.daily_95_avg.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
          </section>

          <section class="mode-panel">
            <div class="mode-panel-header">
              <div>
                <div class="mode-title">月95</div>
                <div class="mode-subtitle">启用且填写流量单价后，月结算任务才会入库</div>
              </div>
              <el-switch v-model="modeForm.range_95.enabled" />
            </div>
            <el-form-item label="结算进制">
              <el-select v-model="modeForm.range_95.unit_base" class="field-w-160" :disabled="!modeForm.range_95.enabled">
                <el-option label="1000" :value="1000" />
                <el-option label="1024" :value="1024" />
              </el-select>
            </el-form-item>
            <el-form-item label="流量单价（节点建设费）">
              <el-input-number v-model="modeForm.range_95.node_construction_fee" :disabled="!modeForm.range_95.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="建设费归属">
              <SearchSelect v-model="modeForm.range_95.node_construction_fee_owner_id" remote clearable :disabled="!modeForm.range_95.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="CP费">
              <el-input-number v-model="modeForm.range_95.cp_fee" :disabled="!modeForm.range_95.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="CP费归属">
              <SearchSelect v-model="modeForm.range_95.cp_fee_owner_id" remote clearable :disabled="!modeForm.range_95.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="机柜费">
              <el-input-number v-model="modeForm.range_95.rack_fee" :disabled="!modeForm.range_95.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="机柜费归属">
              <SearchSelect v-model="modeForm.range_95.rack_fee_owner_id" remote clearable :disabled="!modeForm.range_95.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
            <el-form-item label="其他费">
              <el-input-number v-model="modeForm.range_95.other_fee" :disabled="!modeForm.range_95.enabled" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
            <el-form-item label="其他费归属">
              <SearchSelect v-model="modeForm.range_95.other_fee_owner_id" remote clearable :disabled="!modeForm.range_95.enabled" :remote-method="remoteSearchOwnerUsers" :loading="ownerUserLoading" :options="ownerUserOptions" label-key="label" value-key="id" placeholder="搜索并选择用户" class="field-w-300" @visible-change="onOwnerDropdownVisible" />
            </el-form-item>
          </section>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="groupDialogVisible" :title="groupForm.id ? '编辑结算分组' : '新建结算分组'" width="680px">
      <el-form :model="groupForm" label-width="110px">
        <el-form-item label="分组名称" required>
          <el-input v-model="groupForm.group_name" placeholder="例如 GD-Bilibili" />
        </el-form-item>
        <el-form-item label="区域" required>
          <SearchSelect v-model="groupForm.region" :options="regionOptions" placeholder="选择区域" class="field-w-300" @change="onGroupScopeChange" />
        </el-form-item>
        <el-form-item label="CP" required>
          <SearchSelect v-model="groupForm.cp" :options="cpOptions" placeholder="选择 CP" class="field-w-300" @change="onGroupScopeChange" />
        </el-form-item>
        <el-form-item label="成员节点">
          <SearchSelect
            v-model="groupForm.member_entity_ids"
            multiple
            clearable
            remote
            :remote-method="remoteSearchGroupEntities"
            :loading="groupEntityLoading"
            :options="groupEntityOptions"
            label-key="label"
            value-key="id"
            placeholder="搜索并选择需要合并结算的 EDC 节点"
            class="field-w-420"
            @visible-change="onGroupEntityDropdownVisible"
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="groupForm.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="groupForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialogVisible=false">取消</el-button>
        <el-button type="primary" :loading="groupSaving" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { RateNode, PaginatedData, UpsertRateNodeRequest, EDCNodeSettlementGroup } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { aggregateNodeRateRows, buildNodeRateModePayloads, isNodeRateModeEnabled, normalizeNodeRateMode, type AggregatedNodeRateRow, type NodeRateMode, type NodeRateModeFields, type NodeRateModeForm } from './node-rates-dual-mode'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.node.write'))

const loading = ref(false)
const items = ref<RateNode[]>([])
const displayItems = computed(() => aggregateNodeRateRows(items.value))
const groups = ref<EDCNodeSettlementGroup[]>([])
const groupLoading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const queryCtl = useCancelableQuery()

const query = reactive<{ region?: string; cp?: string; settlement_type?: string }>({})
type EDCEntityOption = {
  id: number
  label: string
  display_name: string
  alias?: string
  region: string
  cp: string
  edc_name?: string
  sn?: string
}
const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])
const entityOptions = ref<EDCEntityOption[]>([])
const entitiesLoading = ref(false)
const ownerUserOptions = ref<{ id: number; label: string }[]>([])
const ownerUserLoading = ref(false)
const userMap = ref<Record<number, { id: number; alias?: string; display_name?: string; username: string }>>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.settlement_type) p.settlement_type = query.settlement_type
  return p
}

async function fetchData(signal?: AbortSignal) {
  loading.value = true
  try {
    const res: PaginatedData<RateNode> = await api.settlementRates.node.list(buildParams(), { signal })
    items.value = res.items || []
    total.value = res.total || 0
    loadUsersForItems()
  } catch (e: any) {
    if (isAbortError(e)) return
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  groupLoading.value = true
  try {
    const params: any = { page: 1, page_size: 1000 }
    if (query.region) params.region = query.region
    if (query.cp) params.cp = query.cp
    const res = await api.settlementRates.nodeGroups.list(params)
    groups.value = res.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载结算分组失败')
  } finally {
    groupLoading.value = false
  }
}

function onSearch() { page.value = 1; queryCtl.run((signal) => fetchData(signal), { toggleIfRunning: true }); loadGroups() }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, settlement_type: undefined }); page.value=1; pageSize.value=10; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }); loadGroups() }
function onPageChange(p: number) { page.value = p; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }

// Dialog
const dialogVisible = ref(false)
const dialogKey = ref(0)
const saving = ref(false)
const form = reactive<Pick<UpsertRateNodeRequest, 'entity_id' | 'display_name' | 'billing_subject_type' | 'billing_subject_id' | 'billing_display_name' | 'region' | 'cp'>>({ region: '', cp: '', billing_subject_type: 'node' })
const selectedEntityID = ref<number | null>(null)
const isGroupRateForm = computed(() => form.billing_subject_type === 'group')
const modeForm = reactive<NodeRateModeForm>({
  daily_95_avg: defaultModeFields(),
  range_95: defaultModeFields(),
})
const groupDialogVisible = ref(false)
const groupSaving = ref(false)
const groupEntityLoading = ref(false)
const groupEntityOptions = ref<EDCEntityOption[]>([])
const groupForm = reactive({
  id: 0,
  group_name: '',
  region: '',
  cp: '',
  enabled: true,
  remark: '',
  member_entity_ids: [] as number[],
})

function defaultModeFields(): NodeRateModeFields {
  return {
    enabled: false,
    had_existing: false,
    unit_base: 1000,
    cp_fee: undefined,
    cp_fee_owner_id: undefined,
    node_construction_fee: undefined,
    node_construction_fee_owner_id: undefined,
    rack_fee: undefined,
    rack_fee_owner_id: undefined,
    other_fee: undefined,
    other_fee_owner_id: undefined,
  }
}

function modeLabel(mode?: string) {
  return mode === 'range_95' || mode === 'monthly95' ? '月95' : '日95均值'
}

function normalizeMode(mode?: string): NodeRateMode {
  return normalizeNodeRateMode(mode)
}

function hasMode(row: AggregatedNodeRateRow, mode: NodeRateMode) {
  return Boolean(row.mode_rates?.[mode]?.node_construction_fee !== undefined && row.mode_rates?.[mode]?.node_construction_fee !== null)
}

function modeRateValue(row: AggregatedNodeRateRow, mode: NodeRateMode, key: keyof RateNode) {
  const value = row.mode_rates?.[mode]?.[key]
  return value === undefined || value === null || value === '' ? '-' : value
}

function ownerSummary(row: AggregatedNodeRateRow) {
  const ids = new Set<number>()
  for (const mode of ['daily_95_avg', 'range_95'] as NodeRateMode[]) {
    const rate = row.mode_rates?.[mode]
    for (const value of [rate?.cp_fee_owner_id, rate?.node_construction_fee_owner_id, rate?.rack_fee_owner_id, rate?.other_fee_owner_id]) {
      const id = Number(value)
      if (Number.isFinite(id) && id > 0) ids.add(id)
    }
  }
  if (ids.size === 0) return '-'
  return Array.from(ids).map((id) => displayOwner(id)).join('、')
}

function ownerTooltipModes(row: AggregatedNodeRateRow) {
  return (['daily_95_avg', 'range_95'] as NodeRateMode[])
    .map((mode) => {
      const rate = row.mode_rates?.[mode]
      return {
        mode,
        label: modeLabel(mode),
        items: [
          { label: 'CP费归属', owner: displayOwner(rate?.cp_fee_owner_id) },
          { label: '流量单价归属', owner: displayOwner(rate?.node_construction_fee_owner_id) },
          { label: '机柜费归属', owner: displayOwner(rate?.rack_fee_owner_id) },
          { label: '其他费归属', owner: displayOwner(rate?.other_fee_owner_id) },
        ],
      }
    })
    .filter((mode) => mode.items.some((item) => item.owner !== '-'))
}

function normalizeEDCEntities(payload: any): EDCEntityOption[] {
  const items = Array.isArray(payload?.items) ? payload.items : []
  return items.map((item: any) => {
    const id = Number(item.id)
    const displayName = String(item.display_name || '')
    const alias = item.alias ? String(item.alias) : ''
    const region = String(item.region || '')
    const cp = String(item.cp || '')
    const edcName = item.edc_name ? String(item.edc_name) : ''
    const sn = item.sn ? String(item.sn) : ''
    const suffix = [region, cp].filter(Boolean).join(' / ')
    const raw = [alias || displayName, edcName, sn].filter(Boolean).join(' · ')
    return {
      id,
      label: suffix ? `${raw || id} (${suffix})` : `${raw || id}`,
      display_name: displayName,
      alias,
      region,
      cp,
      edc_name: edcName,
      sn,
    }
  }).filter((item: EDCEntityOption) => Number.isFinite(item.id) && item.id > 0)
}

function buildUserLabel(user: any): string {
  if (!user) return ''
  const alias = (user.alias && String(user.alias).trim()) ? String(user.alias).trim() : ''
  const displayName = (user.display_name && String(user.display_name).trim()) ? String(user.display_name).trim() : ''
  const username = (user.username && String(user.username).trim()) ? String(user.username).trim() : ''
  const id = Number(user.id)
  return alias || displayName || username || (Number.isFinite(id) ? `用户#${id}` : '')
}

function displayOwner(id?: number | null): string {
  if (!id) return '-'
  const key = Number(id)
  const user = userMap.value[key]
  if (!user) return `用户#${key}`
  return buildUserLabel(user) || `用户#${key}`
}

function collectOwnerIDsFromRates(rates: Array<RateNode | undefined | null>): number[] {
  const ids = new Set<number>()
  for (const row of rates) {
    if (!row) continue
    for (const value of [row.cp_fee_owner_id, row.node_construction_fee_owner_id, row.rack_fee_owner_id, row.other_fee_owner_id]) {
      const id = Number(value)
      if (Number.isFinite(id) && id > 0) ids.add(id)
    }
  }
  return Array.from(ids)
}

function mergeOwnerUserOptions(options: Array<{ id: number; label: string }>) {
  const merged = new Map<number, { id: number; label: string }>()
  for (const option of ownerUserOptions.value) {
    if (Number.isFinite(Number(option.id)) && Number(option.id) > 0) merged.set(Number(option.id), option)
  }
  for (const option of options) {
    if (Number.isFinite(Number(option.id)) && Number(option.id) > 0) merged.set(Number(option.id), option)
  }
  ownerUserOptions.value = Array.from(merged.values())
}

async function loadUsersByIDs(ids: number[], options: { mergeOptions?: boolean } = {}) {
  const unique = Array.from(new Set(ids.filter((id) => Number.isFinite(id) && id > 0)))
  if (unique.length === 0) return
  const missing = unique.filter((id) => !userMap.value[id])
  if (missing.length > 0) {
    const res: any = await api.system.users.list({ ids: missing.join(',') })
    const users: any[] = Array.isArray(res?.items) ? res.items : []
    const next = { ...userMap.value }
    for (const user of users) {
      const id = Number(user.id)
      if (Number.isFinite(id) && id > 0) {
        next[id] = { id, alias: user.alias, display_name: user.display_name, username: user.username }
      }
    }
    userMap.value = next
  }
  if (options.mergeOptions) {
    mergeOwnerUserOptions(unique.map((id) => ({ id, label: displayOwner(id) })).filter((item) => item.label !== '-'))
  }
}

async function loadUsersForItems() {
  const ids = collectOwnerIDsFromRates(items.value)
  if (ids.length === 0) {
    userMap.value = {}
    return
  }
  try {
    await loadUsersByIDs(ids)
  } catch {
    userMap.value = {}
  }
}

async function remoteSearchOwnerUsers(keyword = '') {
  ownerUserLoading.value = true
  try {
    const res: any = await api.system.users.list({ page: 1, page_size: 100, status: 1, username: keyword || undefined })
    const users: any[] = Array.isArray(res?.items) ? res.items : []
    ownerUserOptions.value = users.map((user: any) => ({ id: Number(user.id), label: buildUserLabel(user) })).filter((item) => Number.isFinite(item.id) && item.id > 0)
  } catch {
    ownerUserOptions.value = []
  } finally {
    ownerUserLoading.value = false
  }
}

function onOwnerDropdownVisible(visible: boolean) {
  if (visible) remoteSearchOwnerUsers('')
}

async function loadScopeOptions() {
  try {
    const [regions, cps] = await Promise.all([
      api.v2.edc.getRegions(),
      api.v2.edc.getCPs(),
    ])
    regionOptions.value = Array.isArray(regions) ? regions.filter(Boolean).sort() : []
    cpOptions.value = Array.isArray(cps) ? cps.filter(Boolean).sort() : []
  } catch {
    regionOptions.value = []
    cpOptions.value = []
  }
}

async function remoteSearchEntities(keyword = '') {
  entitiesLoading.value = true
  try {
    const params: any = { limit: 50, offset: 0 }
    if (keyword) params.display_name = keyword
    if (form.region) params.region = form.region
    if (form.cp) params.cp = form.cp
    entityOptions.value = normalizeEDCEntities(await api.v2.edc.getEntities(params))
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载 EDC 节点失败')
    entityOptions.value = []
  } finally {
    entitiesLoading.value = false
  }
}

function clearSelectedEntity() {
  selectedEntityID.value = null
  form.entity_id = undefined
  form.display_name = ''
  form.billing_subject_type = 'node'
  form.billing_subject_id = undefined
  form.billing_display_name = undefined
}

function onEntityScopeChange() {
  clearSelectedEntity()
  if (form.region || form.cp) {
    remoteSearchEntities('')
    return
  }
  entityOptions.value = []
}

function onEntityDropdownVisible(visible: boolean) {
  if (visible) remoteSearchEntities('')
}

function onEntityChange(value: number | null) {
  const selected = entityOptions.value.find((item) => item.id === value)
  if (!selected) {
    form.entity_id = value || undefined
    return
  }
  form.entity_id = selected.id
  form.display_name = selected.display_name
  form.billing_subject_type = 'node'
  form.billing_subject_id = selected.id
  form.billing_display_name = selected.display_name
  form.region = selected.region
  form.cp = selected.cp
}

function onEntityClear() {
  clearSelectedEntity()
}

function resetModeForm() {
  Object.assign(modeForm.daily_95_avg, defaultModeFields())
  Object.assign(modeForm.range_95, defaultModeFields())
}

function applyRateToMode(rate: RateNode) {
  const mode = normalizeMode(rate.settlement_mode || rate.settlement_type)
  Object.assign(modeForm[mode], {
    enabled: rate.node_construction_fee !== undefined && rate.node_construction_fee !== null,
    had_existing: true,
    unit_base: rate.unit_base || 1000,
    cp_fee: rate.cp_fee,
    cp_fee_owner_id: rate.cp_fee_owner_id,
    node_construction_fee: rate.node_construction_fee,
    node_construction_fee_owner_id: rate.node_construction_fee_owner_id,
    rack_fee: rate.rack_fee,
    rack_fee_owner_id: rate.rack_fee_owner_id,
    other_fee: rate.other_fee,
    other_fee_owner_id: rate.other_fee_owner_id,
  })
}

function openDialog(row?: AggregatedNodeRateRow) {
  dialogKey.value += 1
  resetModeForm()
  selectedEntityID.value = row?.entity_id ? Number(row.entity_id) : null
  Object.assign(form, {
    entity_id: row?.entity_id || undefined,
    display_name: row?.display_name || '',
    billing_subject_type: row?.billing_subject_type || 'node',
    billing_subject_id: row?.billing_subject_id || undefined,
    billing_display_name: row?.billing_display_name || row?.display_name || '',
    region: row?.region || '',
    cp: row?.cp || '',
  })
  if (row) {
    const related = Object.values(row.mode_rates || {}).filter(Boolean) as RateNode[]
    for (const item of related.length ? related : [row]) applyRateToMode(item)
    loadUsersByIDs(collectOwnerIDsFromRates(related.length ? related : [row]), { mergeOptions: true })
    if (row.entity_id && row.billing_subject_type !== 'group') {
      entityOptions.value = [{
        id: Number(row.entity_id),
        label: `${row.display_name || row.entity_id} (${row.region} / ${row.cp})`,
        display_name: row.display_name || '',
        region: row.region,
        cp: row.cp,
      }]
    } else {
      entityOptions.value = []
    }
  } else {
    entityOptions.value = []
  }
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp) { ElMessage.warning('区域/CP为必填'); return }
  const enabledModes = [modeForm.daily_95_avg, modeForm.range_95].filter((mode) => mode.enabled)
  if (enabledModes.length === 0) { ElMessage.warning('至少启用一种结算模式'); return }
  if (!enabledModes.some((mode) => isNodeRateModeEnabled(mode))) { ElMessage.warning('至少填写一种结算模式的流量单价'); return }
  if (enabledModes.some((mode) => !isNodeRateModeEnabled(mode))) { ElMessage.warning('启用的结算模式必须填写流量单价'); return }
  const payloads = buildNodeRateModePayloads(form, modeForm)
  saving.value = true
  try {
    await Promise.all(payloads.map((payload) => api.settlementRates.node.upsert(payload)))
    ElMessage.success('保存成功')
    dialogVisible.value = false
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function groupMemberNames(row: EDCNodeSettlementGroup): string[] {
  return (row.members || []).map((member) => String(member.entity?.display_name || member.entity_id)).filter(Boolean)
}

function openDialogFromGroup(row: EDCNodeSettlementGroup) {
  const aggregated = displayItems.value.find((item) => item.billing_subject_type === 'group' && Number(item.billing_subject_id) === Number(row.id))
  if (aggregated) {
    openDialog(aggregated)
    return
  }
  openDialog({
    id: 0,
    entity_id: null,
    display_name: row.group_name,
    billing_subject_type: 'group',
    billing_subject_id: row.id,
    billing_display_name: row.group_name,
    region: row.region,
    cp: row.cp,
    settlement_type: 'daily_95_avg',
    settlement_mode: 'daily_95_avg',
    configured_modes: [],
    mode_rates: {},
  } as AggregatedNodeRateRow)
}

function openGroupDialog(row?: EDCNodeSettlementGroup) {
  Object.assign(groupForm, {
    id: row?.id || 0,
    group_name: row?.group_name || '',
    region: row?.region || '',
    cp: row?.cp || '',
    enabled: row?.enabled ?? true,
    remark: row?.remark || '',
    member_entity_ids: (row?.members || []).map((member) => Number(member.entity_id)).filter((id) => Number.isFinite(id) && id > 0),
  })
  groupEntityOptions.value = (row?.members || [])
    .map((member) => member.entity)
    .filter(Boolean)
    .map((entity: any) => ({
      id: Number(entity.id),
      label: `${entity.display_name || entity.id} (${entity.region || ''} / ${entity.cp || ''})`,
      display_name: entity.display_name || '',
      region: entity.region || '',
      cp: entity.cp || '',
      edc_name: entity.edc_name || '',
      sn: entity.sn || '',
    }))
  groupDialogVisible.value = true
}

function onGroupScopeChange() {
  groupForm.member_entity_ids = []
  groupEntityOptions.value = []
  remoteSearchGroupEntities('')
}

async function remoteSearchGroupEntities(keyword = '') {
  groupEntityLoading.value = true
  try {
    const params: any = { limit: 100, offset: 0 }
    if (keyword) params.display_name = keyword
    if (groupForm.region) params.region = groupForm.region
    if (groupForm.cp) params.cp = groupForm.cp
    groupEntityOptions.value = normalizeEDCEntities(await api.v2.edc.getEntities(params))
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载 EDC 节点失败')
    groupEntityOptions.value = []
  } finally {
    groupEntityLoading.value = false
  }
}

function onGroupEntityDropdownVisible(visible: boolean) {
  if (visible) remoteSearchGroupEntities('')
}

async function saveGroup() {
  if (!groupForm.group_name || !groupForm.region || !groupForm.cp) {
    ElMessage.warning('分组名称、区域、CP为必填')
    return
  }
  groupSaving.value = true
  try {
    const payload = {
      group_name: groupForm.group_name,
      region: groupForm.region,
      cp: groupForm.cp,
      enabled: groupForm.enabled,
      remark: groupForm.remark || null,
      member_entity_ids: groupForm.member_entity_ids,
    }
    if (groupForm.id) await api.settlementRates.nodeGroups.update(groupForm.id, payload)
    else await api.settlementRates.nodeGroups.create(payload)
    ElMessage.success('保存成功')
    groupDialogVisible.value = false
    await loadGroups()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    groupSaving.value = false
  }
}

async function disableGroup(row: EDCNodeSettlementGroup) {
  try {
    await api.settlementRates.nodeGroups.remove(row.id)
    ElMessage.success('已禁用')
    await loadGroups()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '禁用失败')
  }
}

onMounted(() => {
  loadScopeOptions()
  loadGroups()
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
})
usePageRefresh(() => {
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
})
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.mode-tag { margin-left: 6px; }
.rate-subject-name { margin-left: 8px; vertical-align: middle; }
.owner-summary { display: inline-block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; vertical-align: bottom; white-space: nowrap; cursor: default; }
.owner-tooltip { min-width: 220px; display: grid; gap: 10px; }
.owner-tooltip-mode + .owner-tooltip-mode { padding-top: 8px; border-top: 1px solid var(--el-border-color-lighter); }
.owner-tooltip-title { margin-bottom: 6px; font-weight: 600; color: var(--el-text-color-primary); }
.owner-tooltip-row { display: flex; justify-content: space-between; gap: 16px; line-height: 22px; color: var(--el-text-color-regular); }
.owner-tooltip-row strong { font-weight: 500; color: var(--el-text-color-primary); }
.entity-option { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; }
.entity-option-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.entity-option-meta { flex: 0 0 auto; color: var(--el-text-color-secondary); font-size: 12px; }
.node-rate-form { padding-right: 8px; }
.drawer-section { padding-bottom: 10px; border-bottom: 1px solid var(--el-border-color-light); margin-bottom: 14px; }
.section-title { font-weight: 600; color: var(--el-text-color-primary); margin-bottom: 12px; }
.mode-grid { display: grid; gap: 14px; }
.mode-panel { border: 1px solid var(--el-border-color-light); border-radius: 8px; padding: 14px 14px 4px; }
.mode-panel-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 12px; }
.mode-title { font-weight: 600; color: var(--el-text-color-primary); }
.mode-subtitle { margin-top: 2px; font-size: 12px; color: var(--el-text-color-secondary); }
</style>
