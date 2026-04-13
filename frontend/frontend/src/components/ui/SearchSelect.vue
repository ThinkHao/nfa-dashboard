<script setup lang="ts">
import { computed, getCurrentInstance, ref, useAttrs } from 'vue'

import { normalizeSearchOptions } from './search-select-utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  modelValue?: string | number | boolean | string[] | number[] | null
  options?: unknown[]
  labelKey?: string
  valueKey?: string
  loading?: boolean
  placeholder?: string
  clearable?: boolean
  filterable?: boolean
  remote?: boolean
  remoteMethod?: ((query: string) => void | Promise<void>) | undefined
  multiple?: boolean
  reserveKeyword?: boolean
}>(), {
  modelValue: null,
  options: () => [],
  labelKey: 'label',
  valueKey: 'value',
  loading: false,
  placeholder: '',
  clearable: true,
  filterable: true,
  remote: false,
  remoteMethod: undefined,
  multiple: false,
  reserveKeyword: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
  change: [value: any]
  'visible-change': [visible: boolean]
  clear: []
  blur: [event: FocusEvent]
  focus: [event: FocusEvent]
}>()

const attrs = useAttrs()
const selectRef = ref()
const uid = getCurrentInstance()?.uid ?? Math.round(Math.random() * 100000)
const popperClass = `search-select-popper-${uid}`

const normalizedOptions = computed(() => normalizeSearchOptions(props.options, props.labelKey, props.valueKey))

function onModelValueChange(value: any) {
  emit('update:modelValue', value)
  emit('change', value)
}

function onVisibleChange(visible: boolean) {
  emit('visible-change', visible)
}

function onClear() {
  emit('clear')
}

function onBlur(event: FocusEvent) {
  emit('blur', event)
}

function onFocus(event: FocusEvent) {
  emit('focus', event)
}

function triggerOptionSelection(optionEl: HTMLElement | null): boolean {
  if (!optionEl) return false
  optionEl.click()
  return true
}

function pickCandidateOption(): HTMLElement | null {
  const container = document.querySelector(`.${popperClass}`) as HTMLElement | null
  if (!container) return null

  return (
    container.querySelector('.el-select-dropdown__item.hover:not(.is-disabled):not(.is-hidden)') as HTMLElement | null
  ) || (
    container.querySelector('.el-select-dropdown__item:not(.is-disabled):not(.is-hidden)') as HTMLElement | null
  )
}

function handleCommitKey(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== 'Tab') return
  const optionEl = pickCandidateOption()
  if (!optionEl) return

  if (event.key === 'Enter') {
    event.preventDefault()
  }

  triggerOptionSelection(optionEl)
}
</script>

<template>
  <el-select
    ref="selectRef"
    :model-value="modelValue"
    :loading="loading"
    :placeholder="placeholder"
    :clearable="clearable"
    :filterable="filterable"
    :remote="remote"
    :remote-method="remoteMethod"
    :multiple="multiple"
    :reserve-keyword="reserveKeyword"
    :popper-class="popperClass"
    v-bind="attrs"
    @update:model-value="onModelValueChange"
    @visible-change="onVisibleChange"
    @clear="onClear"
    @blur="onBlur"
    @focus="onFocus"
    @keydown.capture="handleCommitKey"
  >
    <el-option
      v-for="option in normalizedOptions"
      :key="`${option.value}`"
      :label="option.label"
      :value="option.value"
    >
      <slot name="option" :option="option.raw" :label="option.label" :value="option.value">
        <span>{{ option.label }}</span>
      </slot>
    </el-option>
  </el-select>
</template>
