<template>
  <div class="formula-tab">
    <el-card class="formula-card">
      <template #header>
        <div class="card-header">
          <div class="card-header__title">
            <h3 class="card-title">结算公式策略</h3>
            <p class="subtitle">拖拽费率字段与运算符，构建可复用的结算金额计算公式</p>
          </div>
          <div class="card-header__actions">
            <el-button
              class="toolbar-btn toolbar-btn--icon"
              type="primary"
              @click="resetToPreset"
              :icon="Refresh"
              circle
              title="恢复默认模板"
            />
            <el-button
              class="toolbar-btn"
              type="primary"
              @click="handleSaveFormula"
              :loading="saving"
              :disabled="!currentFormula"
            >
              保存公式
            </el-button>
          </div>
        </div>
      </template>

      <div class="formula-body" v-loading="loading">
        <div class="formula-nav">
          <el-tabs
            v-model="activeFormulaId"
            type="card"
            class="formula-tabs"
            @tab-add="addFormula"
            @tab-remove="handleRemoveFormula"
            addable
          >
            <el-tab-pane
              v-for="formula in formulas"
              :key="formula.id"
              :name="formula.id"
              :closable="formulas.length > 1"
            >
              <template #label>
                <span>{{ formula.name }}</span>
              </template>
            </el-tab-pane>
          </el-tabs>
        </div>

        <div v-if="currentFormula" class="formula-grid">
          <aside class="palette-area">
            <el-card class="palette-card" shadow="hover">
              <template #header>
                <div class="palette-header">
                  <h4 class="card-title">费率字段</h4>
                  <span class="hint">拖拽字段到右侧公式区</span>
                </div>
              </template>
              <el-scrollbar class="palette-scroll">
                <div
                  v-for="group in fieldGroups"
                  :key="group.key"
                  class="palette-group"
                >
                  <div class="group-title">{{ group.label }}</div>
                  <div class="palette-tags">
                    <el-tooltip
                      v-for="field in group.fields"
                      :key="field.key"
                      :content="field.description"
                      placement="top"
                    >
                      <el-tag
                        class="palette-tag"
                        effect="plain"
                        draggable="true"
                        @dragstart="handlePaletteDragStart($event, fieldPayload(field))"
                      >
                        {{ field.label }}
                      </el-tag>
                    </el-tooltip>
                  </div>
                </div>
              </el-scrollbar>
            </el-card>

            <el-card class="palette-card" shadow="hover">
              <template #header>
                <div class="palette-header">
                  <h4 class="card-title">运算符与常量</h4>
                </div>
              </template>
              <div class="operators">
                <div class="operator-group">
                  <div class="group-title">运算符</div>
                  <div class="palette-tags">
                    <el-tag
                      v-for="op in operatorOptions"
                      :key="op.value"
                      class="palette-tag operator"
                      effect="plain"
                      draggable="true"
                      @dragstart="handlePaletteDragStart($event, operatorPayload(op))"
                    >
                      {{ op.label }}
                    </el-tag>
                  </div>
                </div>
                <div class="operator-group">
                  <div class="group-title">常量</div>
                  <div class="palette-tags">
                    <el-tag
                      v-for="constant in constantPresets"
                      :key="constant.value"
                      class="palette-tag constant"
                      effect="plain"
                      draggable="true"
                      @dragstart="handlePaletteDragStart($event, constantPayload(constant.value))"
                    >
                      {{ constant.label }}
                    </el-tag>
                  </div>
                  <div class="custom-constant">
                    <el-input-number
                      v-model="customConstant"
                      :precision="4"
                      :step="0.1"
                      placeholder="自定义常量"
                      @change="addCustomConstant"
                    />
                  </div>
                </div>
              </div>
            </el-card>
          </aside>

          <section class="builder-area">
            <div class="builder-header">
              <div class="builder-title">
                <el-input
                  v-model="currentFormula.name"
                  class="formula-name-input"
                  size="large"
                  placeholder="请输入公式名称"
                  @change="touchFormula"
                />
                <el-input
                  v-model="currentFormula.description"
                  class="formula-desc-input"
                  placeholder="补充该公式的适用场景或说明"
                  @change="touchFormula"
                />
              </div>
              <div class="builder-actions">
                <el-button
                  class="action-btn"
                  type="primary"
                  @click="clearTokens"
                  :disabled="!currentFormula.tokens.length"
                >
                  清空公式
                </el-button>
              </div>
            </div>

            <div
              class="builder-canvas"
              @dragover.prevent="handleCanvasDragOver"
              @dragenter.prevent="setDragOver(currentFormula.tokens.length)"
              @dragleave="clearDragOver"
              @drop="handleCanvasDrop"
            >
              <div
                class="drop-wrapper"
                :class="{ 'drop-wrapper--empty': !currentFormula.tokens.length }"
              >
                <div
                  class="drop-slot"
                  :class="{ 'is-active': dragOverIndex === 0 }"
                  @dragover.prevent
                  @dragenter.prevent="setDragOver(0)"
                  @dragleave="clearDragOver"
                  @drop="handleDrop($event, 0)"
                ></div>

                <div
                  v-for="(token, index) in currentFormula.tokens"
                  :key="token.id"
                  class="token-wrapper"
                >
                  <el-tag
                    class="token"
                    :type="token.type === 'field' ? 'success' : token.type === 'number' ? 'info' : ''"
                    closable
                    draggable="true"
                    @close="removeToken(token.id)"
                    @dragstart="handleTokenDragStart($event, token.id)"
                    @dragend="clearDragOver"
                  >
                    {{ token.label }}
                  </el-tag>
                  <div
                    class="drop-slot"
                    :class="{ 'is-active': dragOverIndex === index + 1 }"
                    @dragover.prevent
                    @dragenter.prevent="setDragOver(index + 1)"
                    @dragleave="clearDragOver"
                    @drop="handleDrop($event, index + 1)"
                  ></div>
                </div>

                <el-empty v-if="!currentFormula.tokens.length" description="拖拽左侧字段或运算符到此处" />
              </div>
            </div>

            <div class="builder-footer">
              <div class="preview-block">
                <div class="preview-label">公式预览</div>
                <code class="preview-expression">{{ expressionPreview }}</code>
              </div>
              <div class="preview-meta">
                <div class="meta-item">
                  <span class="meta-label">最后编辑</span>
                  <span class="meta-value">{{ formatDate(currentFormula.updated_at) }}</span>
                </div>
                <div class="meta-item">
                  <span class="meta-label">字段数量</span>
                  <span class="meta-value">{{ fieldTokenCount }}</span>
                </div>
              </div>
            </div>
          </section>
        </div>

        <el-empty v-else description="请先新增一个结算公式" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import api from '@/api'
import type {
  CreateSettlementFormulaRequest,
  SettlementFormulaItem,
  SettlementFormulaToken,
  SettlementFormulaTokenType,
  UpdateSettlementFormulaRequest,
} from '@/types/api'

interface FieldOption {
  key: string
  label: string
  value: string
  description: string
}

interface FieldGroup {
  key: string
  label: string
  fields: FieldOption[]
}

interface OperatorOption {
  label: string
  value: string
}

type TokenType = SettlementFormulaTokenType
type FormulaToken = SettlementFormulaToken

interface SettlementFormulaViewModel {
  id: string
  recordId?: number
  name: string
  description?: string
  tokens: FormulaToken[]
  enabled: boolean
  updated_at: string
  create_time?: string
  update_time?: string
  isNew?: boolean
  dirty?: boolean
}

interface PalettePayloadBase {
  origin: 'palette'
  type: TokenType
}

interface PaletteFieldPayload extends PalettePayloadBase {
  type: 'field'
  value: string
  label: string
}

interface PaletteOperatorPayload extends PalettePayloadBase {
  type: 'operator'
  value: string
  label: string
}

interface PaletteConstantPayload extends PalettePayloadBase {
  type: 'number'
  value: string
  label: string
}

type PalettePayload = PaletteFieldPayload | PaletteOperatorPayload | PaletteConstantPayload

interface FormulaDragPayload {
  origin: 'formula'
  tokenId: string
}

type DragPayload = PalettePayload | FormulaDragPayload

const fieldGroups: FieldGroup[] = [
  {
    key: 'core-rate',
    label: '核心费率',
    fields: [
      { key: 'customer_fee', label: '客户费率 customer_fee', value: 'customer_fee', description: 'rate_final_customer.customer_fee' },
      { key: 'network_line_fee', label: '线路费率 network_line_fee', value: 'network_line_fee', description: 'rate_final_customer.network_line_fee' },
      { key: 'node_deduction_fee', label: '节点抵扣 node_deduction_fee', value: 'node_deduction_fee', description: 'rate_final_customer.node_deduction_fee' }
    ]
  },
  {
    key: 'flow-metrics',
    label: '流量指标',
    fields: [
      { key: 'settlement_flow_95', label: '95结算流量', value: 'settlement_flow_95', description: '结算任务生成的 95 峰值流量 (GB)' },
      { key: 'settlement_flow_total', label: '结算总流量', value: 'settlement_flow_total', description: '按结算周期汇总的总流量 (GB)' }
    ]
  },
  {
    key: 'adjustments',
    label: '调整项',
    fields: [
      { key: 'discount_rate', label: '折扣系数 discount_rate', value: 'discount_rate', description: '用于折扣/优惠的系数，默认 1' },
      { key: 'tax_rate', label: '税率 tax_rate', value: 'tax_rate', description: '结算税率，示例值 0.06' },
      { key: 'service_fee', label: '服务费 service_fee', value: 'service_fee', description: '附加的固定服务费' }
    ]
  }
]

const operatorOptions: OperatorOption[] = [
  { label: '+', value: '+' },
  { label: '-', value: '-' },
  { label: '×', value: '*' },
  { label: '÷', value: '/' },
  { label: '(', value: '(' },
  { label: ')', value: ')' }
]

const constantPresets = [
  { label: '0.95', value: '0.95' },
  { label: '1', value: '1' },
  { label: '100', value: '100' },
  { label: '百分比→小数', value: '0.01' }
]

const loading = ref(false)
const saving = ref(false)
const formulas = ref<SettlementFormulaViewModel[]>([])
const activeFormulaId = ref('')
const dragOverIndex = ref<number | null>(null)
const customConstant = ref<number | null>(null)

const currentFormula = computed(() => formulas.value.find((item) => item.id === activeFormulaId.value) || null)

const fieldTokenCount = computed(() => currentFormula.value?.tokens.filter((token) => token.type === 'field').length ?? 0)

const expressionPreview = computed(() => {
  if (!currentFormula.value) return ''
  return currentFormula.value.tokens
    .map((token) => {
      if (token.type === 'field') {
        return `{{${token.value}}}`
      }
      return token.value
    })
    .join(' ')
})

function createId() {
  return `f_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 7)}`
}

function findFieldByKey(key: string) {
  for (const group of fieldGroups) {
    const found = group.fields.find((field) => field.key === key)
    if (found) return found
  }
  return null
}

function createTokenFromPayload(payload: PalettePayload): FormulaToken {
  if (payload.type === 'field') {
    return {
      id: createId(),
      type: 'field',
      value: payload.value,
      label: payload.label
    }
  }
  if (payload.type === 'operator') {
    return {
      id: createId(),
      type: 'operator',
      value: payload.value,
      label: payload.label
    }
  }
  return {
    id: createId(),
    type: 'number',
    value: payload.value,
    label: payload.label
  }
}

function sanitizeTokens(rawTokens: any): FormulaToken[] {
  const source = Array.isArray(rawTokens) ? rawTokens : []
  const results: FormulaToken[] = []
  for (const item of source) {
    if (!item) continue
    const value = typeof item.value === 'string' ? item.value : ''
    if (!value || value === 'final_fee') continue
    let type = (item.type as TokenType) || 'field'
    if (type !== 'field' && type !== 'operator' && type !== 'number') {
      type = value.match(/^[+\-*/()]$/) ? 'operator' : 'field'
    }
    let label = typeof item.label === 'string' && item.label.length ? item.label : ''
    if (!label) {
      const field = findFieldByKey(value)
      label = field?.label || value
    }
    const normalized: FormulaToken = {
      id: typeof item.id === 'string' && item.id.length ? item.id : createId(),
      type,
      value,
      label,
    }
    results.push(normalized)
  }
  return results
}

function createDefaultViewModel(name = '标准公式', presetTokens: FormulaToken[] = []): SettlementFormulaViewModel {
  return {
    id: createId(),
    name,
    description: '请拖拽费率字段与运算符，生成结算公式',
    tokens: presetTokens,
    enabled: true,
    updated_at: new Date().toISOString(),
    isNew: true,
    dirty: true,
  }
}

function toViewModel(item: SettlementFormulaItem): SettlementFormulaViewModel {
  let rawTokens: any = []
  if (typeof item.tokens === 'string') {
    try {
      rawTokens = JSON.parse(item.tokens)
    } catch {
      rawTokens = []
    }
  } else if (Array.isArray(item.tokens)) {
    rawTokens = item.tokens
  }
  const tokens = sanitizeTokens(rawTokens)
  return {
    id: createId(),
    recordId: item.id,
    name: item.name,
    description: item.description || '请拖拽费率字段与运算符，生成结算公式',
    tokens,
    enabled: item.enabled ?? true,
    updated_at: item.update_time || new Date().toISOString(),
    create_time: item.create_time,
    update_time: item.update_time,
    isNew: false,
    dirty: false,
  }
}

function createPresetFormulas(): SettlementFormulaViewModel[] {
  const standard = createDefaultViewModel()
  const sampleTokens = sanitizeTokens([
    { id: createId(), type: 'field', value: 'customer_fee', label: '客户费率 customer_fee' },
    { id: createId(), type: 'operator', value: '*', label: '×' },
    { id: createId(), type: 'field', value: 'settlement_flow_95', label: '95结算流量' },
    { id: createId(), type: 'operator', value: '*', label: '×' },
    { id: createId(), type: 'field', value: 'discount_rate', label: '折扣系数 discount_rate' },
  ])
  const sample = createDefaultViewModel('客户费率 × 流量样例', sampleTokens)
  sample.description = '示例：客户费率 × 95结算流量 × 折扣系数'
  return [standard, sample]
}

function cloneAndSanitizeTokens(tokens: FormulaToken[]): FormulaToken[] {
  return sanitizeTokens(tokens.map((token) => ({ ...token })))
}

async function fetchFormulas() {
  loading.value = true
  try {
    const res = await api.settlement.formulas.list({ limit: 50 })
    const items = res.items.map((item) => toViewModel(item))
    if (items.length) {
      formulas.value = items
      activeFormulaId.value = items[0].id
    } else {
      const defaults = createPresetFormulas()
      formulas.value = defaults
      activeFormulaId.value = defaults[0].id
    }
  } catch (error) {
    console.warn('获取结算公式失败，使用默认模板', error)
    const defaults = createPresetFormulas()
    formulas.value = defaults
    activeFormulaId.value = defaults[0].id
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchFormulas()
})

watch(
  () => formulas.value.length,
  () => {
    if (!formulas.value.length) {
      const defaults = createPresetFormulas()
      formulas.value = defaults
      activeFormulaId.value = defaults[0].id
    } else if (!formulas.value.find((item) => item.id === activeFormulaId.value)) {
      activeFormulaId.value = formulas.value[0].id
    }
  }
)

function addFormula() {
  const name = `自定义公式 ${formulas.value.length + 1}`
  const formula = createDefaultViewModel(name)
  formula.description = '请根据业务需要调整公式内容'
  formulas.value.push(formula)
  activeFormulaId.value = formula.id
}

async function handleSaveFormula() {
  const formula = currentFormula.value
  if (!formula) return
  const name = formula.name.trim()
  if (!name) {
    ElMessage.warning('请输入公式名称')
    return
  }
  if (!formula.tokens.length) {
    ElMessage.warning('请至少添加一个字段或运算符')
    return
  }

  const tokens = cloneAndSanitizeTokens(formula.tokens)
  const description = formula.description?.trim() || undefined
  saving.value = true
  try {
    if (formula.recordId) {
      const payload: UpdateSettlementFormulaRequest = {
        name,
        description,
        tokens,
        enabled: formula.enabled,
      }
      await api.settlement.formulas.update(formula.recordId, payload)
      formula.tokens = tokens
      formula.dirty = false
      formula.isNew = false
      formula.update_time = new Date().toISOString()
      formula.updated_at = formula.update_time
      ElMessage.success('公式已保存')
    } else {
      const payload: CreateSettlementFormulaRequest = {
        name,
        description,
        tokens,
        enabled: formula.enabled,
      }
      const created = await api.settlement.formulas.create(payload)
      const viewModel = toViewModel(created)
      formula.recordId = viewModel.recordId
      formula.create_time = viewModel.create_time
      formula.update_time = viewModel.update_time
      formula.tokens = tokens
      formula.enabled = viewModel.enabled
      formula.dirty = false
      formula.isNew = false
      formula.updated_at = viewModel.updated_at
      ElMessage.success('公式已保存')
    }
  } catch (error) {
    console.warn('保存结算公式失败', error)
    ElMessage.error('保存失败，请稍后重试')
  } finally {
    saving.value = false
  }
}

async function handleRemoveFormula(id: string) {
  if (formulas.value.length <= 1) {
    ElMessage.warning('至少保留一个公式配置')
    return
  }
  const target = formulas.value.find((item) => item.id === id)
  if (!target) return

  if (target.recordId) {
    try {
      await ElMessageBox.confirm('确认删除该结算公式吗？操作将不可恢复。', '删除确认', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      })
      await api.settlement.formulas.remove(target.recordId)
      ElMessage.success('删除成功')
    } catch (error) {
      if (error !== 'cancel') {
        console.warn('删除结算公式失败', error)
        ElMessage.error('删除失败，请稍后重试')
      }
      return
    }
  }

  formulas.value = formulas.value.filter((item) => item.id !== id)
  if (formulas.value.length) {
    activeFormulaId.value = formulas.value[0].id
  } else {
    const defaults = createPresetFormulas()
    formulas.value = defaults
    activeFormulaId.value = defaults[0].id
  }
}

function handlePaletteDragStart(event: DragEvent, payload: PalettePayload) {
  event.dataTransfer?.setData('application/json', JSON.stringify(payload))
  event.dataTransfer?.setData('text/plain', payload.type)
  event.dataTransfer!.effectAllowed = 'copyMove'
}

function handleTokenDragStart(event: DragEvent, tokenId: string) {
  const payload: FormulaDragPayload = { origin: 'formula', tokenId }
  event.dataTransfer?.setData('application/json', JSON.stringify(payload))
  event.dataTransfer?.setData('text/plain', 'formula-token')
  event.dataTransfer!.effectAllowed = 'move'
}

function handleDrop(event: DragEvent, targetIndex: number) {
  event.preventDefault()
  event.stopPropagation()
  clearDragOver()
  const data = event.dataTransfer?.getData('application/json')
  if (!data || !currentFormula.value) return

  try {
    const payload = JSON.parse(data) as DragPayload
    if (payload.origin === 'formula') {
      const sourceIndex = currentFormula.value.tokens.findIndex((token) => token.id === payload.tokenId)
      if (sourceIndex === -1) return
      const [moving] = currentFormula.value.tokens.splice(sourceIndex, 1)
      let insertIndex = targetIndex
      if (sourceIndex < targetIndex) {
        insertIndex -= 1
      }
      currentFormula.value.tokens.splice(insertIndex, 0, moving)
      touchFormula()
      return
    }

    const token = createTokenFromPayload(payload)
    currentFormula.value.tokens.splice(targetIndex, 0, token)
    touchFormula()
  } catch (error) {
    console.warn('解析拖拽数据失败', error)
  }
}

function removeToken(tokenId: string) {
  if (!currentFormula.value) return
  currentFormula.value.tokens = currentFormula.value.tokens.filter((token) => token.id !== tokenId)
  touchFormula()
}

function clearTokens() {
  if (!currentFormula.value) return
  currentFormula.value.tokens = []
  touchFormula()
}

function touchFormula() {
  if (!currentFormula.value) return
  currentFormula.value.updated_at = new Date().toISOString()
  currentFormula.value.dirty = true
}

function setDragOver(index: number) {
  dragOverIndex.value = index
}

function clearDragOver() {
  dragOverIndex.value = null
}

function handleCanvasDrop(event: DragEvent) {
  if (!currentFormula.value) return
  handleDrop(event, currentFormula.value.tokens.length)
}

function handleCanvasDragOver(event: DragEvent) {
  event.dataTransfer && (event.dataTransfer.dropEffect = 'copy')
  if (!currentFormula.value) return
  if (dragOverIndex.value === null) {
    setDragOver(currentFormula.value.tokens.length)
  }
}

function formatDate(value: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未记录'
  return date.toLocaleString('zh-CN')
}

function fieldPayload(field: FieldOption): PaletteFieldPayload {
  return { origin: 'palette', type: 'field', value: field.value, label: field.label }
}

function operatorPayload(op: OperatorOption): PaletteOperatorPayload {
  return { origin: 'palette', type: 'operator', value: op.value, label: op.label }
}

function constantPayload(value: string): PaletteConstantPayload {
  return { origin: 'palette', type: 'number', value, label: value }
}

function addCustomConstant() {
  if (customConstant.value === null || customConstant.value === undefined) return
  const value = Number(customConstant.value)
  if (Number.isNaN(value)) return
  const label = value.toString()
  customConstant.value = null
  if (!currentFormula.value) return
  const token = createTokenFromPayload({ origin: 'palette', type: 'number', value: label, label })
  currentFormula.value.tokens.push(token)
  touchFormula()
}

async function resetToPreset() {
  try {
    await ElMessageBox.confirm('确定恢复为系统默认的模板吗？当前配置将会被覆盖。', '恢复默认', {
      confirmButtonText: '恢复',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const defaults = createPresetFormulas()
    formulas.value = defaults
    activeFormulaId.value = defaults[0].id
    ElMessage.success('已恢复默认模板')
  } catch (error) {
    if (error !== 'cancel') {
      console.warn('恢复默认模板失败', error)
    }
  }
}
</script>

<style scoped>
.formula-tab {
  padding: 12px;
}

.formula-card {
  border-radius: 24px;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.card-header__title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}

.card-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-btn {
  height: 40px;
  padding: 0 18px;
  border-radius: 20px;
  font-weight: 600;
  letter-spacing: 0.4px;
  box-shadow: 0 12px 30px rgba(37, 99, 235, 0.25);
}

.toolbar-btn:hover {
  box-shadow: 0 16px 40px rgba(37, 99, 235, 0.35);
}

.toolbar-btn--icon {
  width: 42px;
  min-width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  padding: 0;
}

.outline-btn {
  padding: 0 16px;
  height: 38px;
  border-radius: 18px;
  font-weight: 600;
  letter-spacing: 0.3px;
  border-color: rgba(59, 130, 246, 0.55);
  color: rgba(37, 99, 235, 0.92);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.35);
}

.outline-btn:hover {
  border-color: rgba(37, 99, 235, 0.85);
  color: rgba(30, 64, 175, 1);
}

.action-btn {
  padding: 0 20px;
  height: 42px;
  border-radius: 20px;
  font-weight: 700;
  letter-spacing: 0.5px;
  background-image: linear-gradient(120deg, #1d4ed8, #2563eb, #3b82f6);
  box-shadow: 0 16px 42px rgba(37, 99, 235, 0.28);
}

.action-btn:hover {
  filter: brightness(1.05);
  box-shadow: 0 20px 52px rgba(37, 99, 235, 0.38);
}

.action-btn.is-disabled,
.action-btn.is-disabled:hover {
  background-image: none;
  background-color: rgba(148, 163, 184, 0.35);
  color: rgba(51, 65, 85, 0.65);
  box-shadow: none;
  cursor: not-allowed;
}

.formula-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.formula-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.formula-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.formula-grid {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 18px;
  align-items: stretch;
}

.palette-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.palette-card {
  border-radius: 18px;
}

.palette-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.hint {
  font-size: 12px;
  color: var(--text-muted);
}

.palette-scroll {
  max-height: 360px;
}

.palette-group {
  margin-bottom: 16px;
}

.palette-group:last-child {
  margin-bottom: 0;
}

.group-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 8px;
}

.palette-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.palette-tag {
  cursor: grab;
  user-select: none;
}

.operator {
  border-color: rgba(37, 99, 235, 0.32);
}

.constant {
  border-color: rgba(14, 165, 233, 0.28);
}

.custom-constant {
  margin-top: 12px;
  display: flex;
}

.builder-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.builder-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.builder-title {
  flex: 1;
  min-width: 280px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.formula-name-input :deep(.el-input__wrapper) {
  border-radius: 14px;
  padding: 6px 14px;
  font-size: 18px;
  font-weight: 600;
}

.formula-desc-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  padding: 4px 12px;
}


.builder-canvas {
  min-height: 220px;
  border-radius: 18px;
  background: rgba(59, 130, 246, 0.05);
  border: 1px dashed rgba(59, 130, 246, 0.25);
  padding: 14px;
  display: flex;
  align-items: center;
}

.drop-wrapper {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.drop-wrapper--empty {
  justify-content: center;
  align-items: center;
  flex-direction: column;
  gap: 12px;
  min-height: 160px;
}

.drop-wrapper--empty .drop-slot {
  display: none;
}

.drop-wrapper--empty :deep(.el-empty) {
  pointer-events: none;
}

.drop-slot {
  width: 18px;
  height: 42px;
  border-radius: 12px;
  border: 2px dashed transparent;
  transition: border-color 0.2s ease, background-color 0.2s ease;
}

.drop-slot.is-active {
  border-color: rgba(37, 99, 235, 0.55);
  background: rgba(59, 130, 246, 0.12);
}

.token-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
}

.token {
  cursor: grab;
  user-select: none;
  border-radius: 12px;
  padding: 6px 12px;
}

.builder-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.preview-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-label {
  font-size: 13px;
  color: var(--text-muted);
}

.preview-expression {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  background: rgba(15, 23, 42, 0.06);
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 14px;
}

.preview-meta {
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.meta-label {
  font-size: 12px;
  color: var(--text-muted);
}

.meta-value {
  font-size: 14px;
  font-weight: 600;
}

@media (max-width: 1280px) {
  .formula-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .palette-scroll {
    max-height: 260px;
  }
  .builder-canvas {
    min-height: 180px;
  }
}
</style>
