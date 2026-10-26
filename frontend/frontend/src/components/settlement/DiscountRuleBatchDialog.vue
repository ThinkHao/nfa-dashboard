<template>
  <el-dialog
    :model-value="modelValue"
    title="批量设置折损规则"
    width="1080px"
    :close-on-click-modal="!submitting"
    :close-on-press-escape="!submitting"
    :show-close="!submitting"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="可以按省份设置默认规则，也可以为同省院校分组设置不同规则；院校规则优先于省份规则。"
      class="mb-12"
    />

    <el-form label-position="top">
      <el-form-item label="院校候选省份">
        <SearchSelect
          v-model="selectedRegions"
          multiple
          collapse-tags
          collapse-tags-tooltip
          :options="regionOptions"
          placeholder="选择院校所在省份后加载院校"
          class="field-w-full"
        />
      </el-form-item>
    </el-form>
    <el-alert
      v-if="missingRequestedRegions.length"
      type="warning"
      :closable="false"
      show-icon
      :title="`当前区域选项未识别到：${missingRequestedRegions.join('、')}。请核对区域名称或当前可见范围。`"
      class="mb-12"
    />

    <div class="groups-toolbar">
      <div>
        <strong>规则组</strong>
        <span class="muted">每组可绑定多个省份或多所院校</span>
      </div>
      <el-button type="primary" plain @click="addGroup">添加规则组</el-button>
    </div>

    <el-empty v-if="groups.length === 0" description="请添加规则组" :image-size="72" />
    <div v-else class="groups-list">
      <el-card v-for="(group, index) in groups" :key="group.localId" shadow="never" class="group-card">
        <template #header>
          <div class="group-header">
            <span>规则组 {{ index + 1 }}</span>
            <el-button type="danger" link @click="removeGroup(index)">移除</el-button>
          </div>
        </template>

        <div class="group-grid">
          <el-form-item label="规则名称" required>
            <el-input v-model="group.name" placeholder="例如：陕西重点院校折损" />
          </el-form-item>
          <el-form-item label="适用层级" required>
            <el-select v-model="group.scopeType" class="field-w-full">
              <el-option label="省份默认" value="region" />
              <el-option label="指定院校" value="school" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="group.priority" :min="0" :step="1" controls-position="right" class="field-w-full" />
          </el-form-item>
        </div>

        <el-form-item v-if="group.scopeType === 'region'" label="适用省份" required>
          <SearchSelect
            v-model="group.regions"
            multiple
            collapse-tags
            collapse-tags-tooltip
            :options="regionOptions"
            placeholder="选择一个或多个省份"
            class="field-w-full"
          />
        </el-form-item>
        <el-form-item v-else label="适用院校" required>
          <SearchSelect
            v-model="group.schoolKeys"
            multiple
            collapse-tags
            collapse-tags-tooltip
            :options="schoolOptions"
            :loading="schoolsLoading"
            placeholder="先选择上方的院校候选省份，再选择院校"
            class="field-w-full"
          />
        </el-form-item>

        <el-form-item label="作用字段">
          <el-checkbox-group v-model="group.fields" class="field-group">
            <el-checkbox label="customer_fee">客户费率</el-checkbox>
            <el-checkbox label="network_line_fee">线路费率</el-checkbox>
            <el-checkbox label="general_fee">节点通用费率</el-checkbox>
            <el-checkbox label="channel_rate">渠道费率</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <div class="items-toolbar">
          <span class="items-label">服务年限折损</span>
          <div class="toolbar-actions">
            <el-button size="small" @click="fillStandard(group)">快速填充 100%/75%/25%</el-button>
            <el-button size="small" @click="addItem(group)">新增区间</el-button>
          </div>
        </div>
        <el-table :data="group.items" border size="small" max-height="220">
          <el-table-column label="开始年" width="150">
            <template #default="{ row }">
              <el-input-number v-model="row.from_year" size="small" :min="1" :step="1" controls-position="right" />
            </template>
          </el-table-column>
          <el-table-column label="结束年" width="210">
            <template #default="{ row }">
              <div class="end-year-cell">
                <el-input-number v-model="row.to_year" size="small" :min="row.from_year || 1" :step="1" controls-position="right" :disabled="row.to_year == null" />
                <el-button size="small" text type="primary" @click="row.to_year = row.to_year == null ? row.from_year : null">
                  {{ row.to_year == null ? '设置上限' : '无上限' }}
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="折损系数" width="230">
            <template #default="{ row }">
              <div class="discount-cell">
                <el-input-number v-model="row.discount_rate" size="small" :min="0.01" :max="1" :step="0.01" :precision="2" controls-position="right" />
                <span>{{ Math.round((Number(row.discount_rate) || 0) * 100) }}%</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ $index }">
              <el-button text type="danger" @click="group.items.splice($index, 1)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <div v-if="preview" class="preview-panel">
      <div class="groups-toolbar">
        <strong>影响预览</strong>
        <el-tag :type="preview.can_apply ? 'success' : 'danger'">
          {{ preview.can_apply ? '可以应用' : '存在范围冲突' }}
        </el-tag>
      </div>
      <el-alert
        v-if="!preview.can_apply"
        type="warning"
        :closable="false"
        show-icon
        title="同层级适用范围已存在规则。请先调整现有规则或移除冲突项，再重新预览。"
        class="mb-8"
      />
      <el-table :data="preview.groups" border size="small">
        <el-table-column prop="name" label="规则组" min-width="180" />
        <el-table-column label="目标数" width="100">
          <template #default="{ row }">{{ row.target_count }}</template>
        </el-table-column>
        <el-table-column label="命中院校" width="110">
          <template #default="{ row }">{{ row.matched_school_count }}</template>
        </el-table-column>
        <el-table-column label="未匹配目标" min-width="190">
          <template #default="{ row }">
            <el-tooltip v-if="row.unmatched_scopes?.length" :content="formatUnmatched(row.unmatched_scopes)" placement="top">
              <el-tag type="warning">{{ row.unmatched_scopes.length }} 项</el-tag>
            </el-tooltip>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="同级冲突" min-width="220">
          <template #default="{ row }">
            <span v-if="row.conflicts?.length" class="conflict-text">
              {{ formatConflicts(row.conflicts) }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
      <p class="preview-note">院校级规则会覆盖同省默认规则；未匹配到当前费率的院校仍会保存为目标，并在有费率记录后生效。</p>
    </div>

    <template #footer>
      <el-button :disabled="submitting" @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" plain :loading="previewing" :disabled="submitting || previewing || groups.length === 0" @click="runPreview">预览影响</el-button>
      <el-button type="primary" :loading="submitting" :disabled="previewing || !preview?.can_apply || previewStale" @click="applyBatch">确认保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import SearchSelect from '@/components/ui/SearchSelect.vue'

type DiscountItem = { from_year: number; to_year: number | null; discount_rate: number }
type RuleGroup = {
  localId: number
  name: string
  scopeType: 'region' | 'school'
  priority: number
  regions: string[]
  schoolKeys: string[]
  fields: string[]
  items: DiscountItem[]
}

const props = defineProps<{ modelValue: boolean; regionOptions: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; created: [] }>()

const selectedRegions = ref<string[]>([])
const groups = ref<RuleGroup[]>([])
const schoolTargets = ref<any[]>([])
const schoolsLoading = ref(false)
const previewing = ref(false)
const submitting = ref(false)
const preview = ref<any>(null)
const previewStale = ref(false)
let nextLocalId = 1
const requestedProvinces = ['陕西', '河北', '安徽', '吉林', '辽宁', '山西']

function normalizeRegionName(value: string) {
  return String(value || '').replace(/省|市|自治区|特别行政区/g, '').trim()
}

const matchedRequestedRegions = computed(() => {
  const options = new Map(props.regionOptions.map((region) => [normalizeRegionName(region), region]))
  return requestedProvinces.map((region) => options.get(normalizeRegionName(region))).filter((region): region is string => !!region)
})
const missingRequestedRegions = computed(() => {
  const matched = new Set(matchedRequestedRegions.value.map(normalizeRegionName))
  return requestedProvinces.filter((region) => !matched.has(normalizeRegionName(region)))
})

const schoolOptions = computed(() => schoolTargets.value.map((item) => {
  const region = String(item?.region || '')
  const cp = String(item?.cp || '')
  const schoolName = String(item?.school_name || '')
  const value = JSON.stringify([region, cp, schoolName])
  return { label: `${region} / ${cp} / ${schoolName}`, value }
}))

function createGroup(scopeType: 'region' | 'school' = 'school'): RuleGroup {
  return {
    localId: nextLocalId++,
    name: `折损规则组 ${groups.value.length + 1}`,
    scopeType,
    priority: 100,
    regions: [],
    schoolKeys: [],
    fields: ['customer_fee'],
    items: [
      { from_year: 1, to_year: 1, discount_rate: 1 },
      { from_year: 2, to_year: 2, discount_rate: 0.75 },
      { from_year: 3, to_year: null, discount_rate: 0.25 },
    ],
  }
}

function addGroup() {
  const scopeType = groups.value.length === 0 ? 'region' : 'school'
  const group = createGroup(scopeType)
  if (scopeType === 'region') group.regions = [...selectedRegions.value]
  groups.value.push(group)
}
function removeGroup(index: number) { groups.value.splice(index, 1) }

function addItem(group: RuleGroup) {
  const last = group.items[group.items.length - 1]
  const fromYear = last && last.to_year != null ? last.to_year + 1 : last ? last.from_year + 1 : 1
  group.items.push({ from_year: fromYear, to_year: fromYear, discount_rate: 1 })
}

function fillStandard(group: RuleGroup) {
  group.items = [
    { from_year: 1, to_year: 1, discount_rate: 1 },
    { from_year: 2, to_year: 2, discount_rate: 0.75 },
    { from_year: 3, to_year: null, discount_rate: 0.25 },
  ]
}

async function loadSchoolTargets() {
  if (selectedRegions.value.length === 0) {
    schoolTargets.value = []
    groups.value.forEach((group) => { group.schoolKeys = [] })
    return
  }
  schoolsLoading.value = true
  try {
    const targetsByRegion = await Promise.all(selectedRegions.value.map(async (region) => {
      const regionTargets: any[] = []
      let offset = 0
      let total = Number.POSITIVE_INFINITY
      const pageSize = 500
      while (offset < total) {
        const result: any = await api.v2.getSchools({ region, limit: pageSize, offset })
        const items = Array.isArray(result?.items) ? result.items : Array.isArray(result) ? result : []
        total = Number(result?.total ?? items.length)
        regionTargets.push(...items)
        if (items.length === 0) break
        offset += items.length
      }
      return regionTargets
    }))
    const targets = targetsByRegion.flat()
    const unique = new Map<string, any>()
    for (const item of targets) {
      const target = {
        region: String(item?.region || ''),
        cp: String(item?.cp || ''),
        school_name: String(item?.school_name || ''),
      }
      if (!target.region || !target.cp || !target.school_name) continue
      unique.set(JSON.stringify([target.region, target.cp, target.school_name]), target)
    }
    schoolTargets.value = Array.from(unique.values())
    const allowed = new Set(schoolOptions.value.map((item) => item.value))
    groups.value.forEach((group) => {
      group.schoolKeys = group.schoolKeys.filter((key) => allowed.has(key))
    })
  } catch (error: any) {
    schoolTargets.value = []
    ElMessage.error(error?.response?.data?.message || error?.message || '加载院校范围失败')
  } finally {
    schoolsLoading.value = false
  }
}

function buildPayload() {
  return {
    groups: groups.value.map((group) => {
      const scopes = group.scopeType === 'region'
        ? group.regions.map((region) => ({ scope_type: 'region', scope_key: region, region }))
        : group.schoolKeys.map((key) => {
          const [region, cp, school_name] = JSON.parse(key)
          return { scope_type: 'school', region, cp, school_name }
        })
      return {
        rule: {
          name: group.name,
          scope_type: group.scopeType,
          enabled: true,
          priority: Number(group.priority),
          fields: group.fields.length > 0 ? group.fields : ['customer_fee'],
        },
        scopes,
        items: group.items.map((item) => ({
          from_year: Number(item.from_year),
          to_year: item.to_year == null ? null : Number(item.to_year),
          discount_rate: Number(item.discount_rate),
        })),
      }
    }),
  }
}

function validateForm(): boolean {
  if (groups.value.length === 0) {
    ElMessage.warning('请至少添加一个规则组')
    return false
  }
  for (let i = 0; i < groups.value.length; i++) {
    const group = groups.value[i]
    if (!group.name.trim()) {
      ElMessage.warning(`规则组 ${i + 1} 请填写名称`)
      return false
    }
    if (group.scopeType === 'region' && group.regions.length === 0) {
      ElMessage.warning(`规则组 ${i + 1} 请至少选择一个省份`)
      return false
    }
    if (group.scopeType === 'school' && group.schoolKeys.length === 0) {
      ElMessage.warning(`规则组 ${i + 1} 请至少选择一所院校`)
      return false
    }
    if (group.items.length === 0) {
      ElMessage.warning(`规则组 ${i + 1} 请至少添加一条折损区间`)
      return false
    }
    const ordered = [...group.items].sort((a, b) => Number(a.from_year) - Number(b.from_year))
    let previousEnd: number | null = null
    for (let j = 0; j < ordered.length; j++) {
      const item = ordered[j]
      const from = Number(item.from_year)
      const to = item.to_year == null ? null : Number(item.to_year)
      const rate = Number(item.discount_rate)
      if (!Number.isInteger(from) || from < 1 || (to != null && (!Number.isInteger(to) || to < from))) {
        ElMessage.warning(`规则组 ${i + 1} 的第 ${j + 1} 条区间无效`)
        return false
      }
      if (previousEnd == null && j > 0 || previousEnd != null && from <= previousEnd) {
        ElMessage.warning(`规则组 ${i + 1} 的折损区间重叠`)
        return false
      }
      if (!Number.isFinite(rate) || rate <= 0 || rate > 1) {
        ElMessage.warning(`规则组 ${i + 1} 的折损系数必须大于 0 且不超过 1`)
        return false
      }
      previousEnd = to
    }
  }
  return true
}

async function runPreview() {
  if (!validateForm()) return
  const payload = buildPayload()
  const requestSnapshot = JSON.stringify(payload)
  previewing.value = true
  previewStale.value = true
  try {
    preview.value = await api.settlementRates.discountRules.previewBatch(payload)
    previewStale.value = requestSnapshot !== JSON.stringify(buildPayload())
  } catch (error: any) {
    preview.value = null
    ElMessage.error(error?.response?.data?.message || error?.message || '预览失败')
  } finally {
    previewing.value = false
  }
}

function formatUnmatched(scopes: any[]) {
  return scopes.map((item) => `${item.region || ''} / ${item.cp || ''} / ${item.school_name || ''}`).join('；')
}

function formatConflicts(rules: any[]) {
  return rules.map((conflict) => {
    const scope = conflict.existing_scope || {}
    const target = scope.scope_type === 'region'
      ? scope.scope_key || scope.region
      : [scope.region, scope.cp, scope.school_name].filter(Boolean).join(' / ')
    return `#${conflict.rule_id} ${conflict.rule_name}（${conflict.enabled ? '启用' : '停用'}，${target}）`
  }).join('；')
}

async function applyBatch() {
  if (!preview.value?.can_apply || previewStale.value || !validateForm()) return
  try {
    await ElMessageBox.confirm('确认按以上预览创建所有规则组？本次创建将作为一个整体提交。', '确认批量保存', {
      type: 'warning',
      confirmButtonText: '确认保存',
      cancelButtonText: '返回检查',
    })
  } catch { return }
  submitting.value = true
  try {
    await api.settlementRates.discountRules.createBatch(buildPayload())
    ElMessage.success('批量规则已保存')
    emit('update:modelValue', false)
    emit('created')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '批量保存失败')
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  selectedRegions.value = [...matchedRequestedRegions.value]
  groups.value = []
  schoolTargets.value = []
  preview.value = null
  previewStale.value = false
  nextLocalId = 1
}

watch(() => props.modelValue, (visible) => {
  if (visible) resetForm()
})
watch(selectedRegions, loadSchoolTargets)
watch(groups, () => {
  previewStale.value = true
}, { deep: true })
</script>

<style scoped>
.groups-toolbar,
.group-header,
.items-toolbar,
.toolbar-actions,
.end-year-cell,
.discount-cell {
  display: flex;
  align-items: center;
}

.groups-toolbar,
.group-header,
.items-toolbar {
  justify-content: space-between;
  gap: 12px;
}

.groups-toolbar { margin: 10px 0; }
.groups-list { display: grid; gap: 12px; max-height: 52vh; overflow: auto; padding-right: 4px; }
.group-card { border-color: var(--el-border-color-lighter); }
.group-grid { display: grid; grid-template-columns: minmax(220px, 2fr) minmax(170px, 1fr) minmax(150px, .7fr); gap: 12px; }
.muted,
.preview-note { color: var(--text-muted); font-size: 12px; }
.items-toolbar { margin: 8px 0; }
.items-label { font-weight: 600; }
.toolbar-actions { gap: 8px; }
.end-year-cell,
.discount-cell { gap: 8px; }
.discount-cell span { color: var(--text-muted); min-width: 42px; }
.preview-panel { margin-top: 14px; }
.preview-note { margin: 8px 0 0; }
.conflict-text { color: var(--el-color-danger); }

@media (max-width: 720px) {
  .group-grid { grid-template-columns: 1fr; gap: 0; }
}
</style>
