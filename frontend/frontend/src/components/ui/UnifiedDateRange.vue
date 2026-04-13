<script setup lang="ts">
import { computed } from 'vue'
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

function onChange(value: [string, string] | null) {
  const normalized = normalizeRangeValue(value as RangeValue, props.type, props.valueFormat)
  emit('update:modelValue', normalized)
  emit('change', normalized)
}
</script>

<template>
  <el-date-picker
    :model-value="modelValue"
    :type="type"
    range-separator="至"
    :start-placeholder="resolvedStartPlaceholder"
    :end-placeholder="resolvedEndPlaceholder"
    :format="format"
    :value-format="valueFormat"
    :default-time="defaultTime"
    class="field-w-300 unified-date-range"
    @update:model-value="onChange"
    @change="onChange"
  />
</template>
