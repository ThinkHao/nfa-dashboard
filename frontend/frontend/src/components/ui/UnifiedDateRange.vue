<script setup lang="ts">
import { computed, ref } from 'vue'
import { normalizeRangeValue, type RangeValue, type UnifiedRangeType } from './unified-date-range-utils'

const props = withDefaults(defineProps<{
  modelValue?: [string, string] | null
  type?: UnifiedRangeType
  format?: string
  valueFormat?: string
  startPlaceholder?: string
  endPlaceholder?: string
}>(), {
  modelValue: null,
  type: 'daterange',
  format: 'YYYY-MM-DD',
  valueFormat: 'YYYY-MM-DD',
  startPlaceholder: '',
  endPlaceholder: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: [string, string] | null]
  change: [value: [string, string] | null]
}>()

const resolvedStartPlaceholder = computed(() => {
  if (props.startPlaceholder) return props.startPlaceholder
  if (props.type === 'monthrange') return '开始月份'
  if (props.type === 'datetimerange') return '开始时间'
  return '开始日期'
})

const resolvedEndPlaceholder = computed(() => {
  if (props.endPlaceholder) return props.endPlaceholder
  if (props.type === 'monthrange') return '结束月份'
  if (props.type === 'datetimerange') return '结束时间'
  return '结束日期'
})

const defaultTime = computed(() => {
  if (props.type === 'monthrange') return undefined
  return [new Date(2000, 0, 1, 0, 0, 0), new Date(2000, 0, 1, 23, 59, 59)]
})

const pickerType = computed(() => props.type === 'datetimerange' ? 'daterange' : props.type)
const pendingStartDate = ref<Date | null>(null)

function resolveShortcutStartDate(): Date {
  if (pendingStartDate.value) return pendingStartDate.value
  const currentStart = props.modelValue?.[0]
  if (!currentStart) return new Date()
  const timestamp = /^\d+$/.test(currentStart) ? Number(currentStart) : currentStart.replace(' ', 'T')
  const parsed = new Date(timestamp)
  return Number.isNaN(parsed.getTime()) ? new Date() : parsed
}

const hasShortcutStartDate = computed(() => Boolean(pendingStartDate.value || props.modelValue?.[0]))

const shortcuts = computed(() => {
  if (props.type === 'monthrange') return []
  if (!hasShortcutStartDate.value) {
    return [{ text: '先选开始日期', value: () => undefined }]
  }

  const start = resolveShortcutStartDate()
  const startDay = new Date(start.getFullYear(), start.getMonth(), start.getDate())
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  const previousMonthEnd = new Date(today.getFullYear(), today.getMonth(), 0)

  return [
    { text: '结束到今天', date: today },
    { text: '结束到昨天', date: yesterday },
    { text: '结束到上月底', date: previousMonthEnd },
  ]
    .filter(({ date }) => date >= startDay)
    .map(({ text, date }) => ({
      text,
      value: () => [start, date],
    }))
})

const pickerValue = computed(() => normalizeRangeValue(props.modelValue as RangeValue, props.type, props.valueFormat))

function normalizePickerValue(value: [string, string] | null) {
  return normalizeRangeValue(value as RangeValue, props.type, props.valueFormat)
}

function onPickerModelUpdate(value: [string, string] | null) {
  const normalized = normalizePickerValue(value)
  pendingStartDate.value = null
  emit('update:modelValue', normalized)
}

function onCalendarChange(value: [Date | null, Date | null]) {
  pendingStartDate.value = value?.[0] instanceof Date ? value[0] : null
}

function onPickerChange(value: [string, string] | null) {
  const normalized = normalizePickerValue(value)
  emit('change', normalized)
}
</script>

<template>
  <el-date-picker
    :model-value="pickerValue"
    :type="pickerType"
    range-separator="至"
    :start-placeholder="resolvedStartPlaceholder"
    :end-placeholder="resolvedEndPlaceholder"
    :format="format"
    :value-format="valueFormat"
    :default-time="defaultTime"
    :shortcuts="shortcuts"
    class="field-w-300 unified-date-range"
    @update:model-value="onPickerModelUpdate"
    @calendar-change="onCalendarChange"
    @change="onPickerChange"
  />
</template>
