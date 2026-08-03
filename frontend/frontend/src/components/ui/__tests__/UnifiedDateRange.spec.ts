import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'

import UnifiedDateRange from '../UnifiedDateRange.vue'

describe('UnifiedDateRange', () => {
  it('uses an immediate date-range panel and sets only the range end through the Today shortcut', async () => {
    let receivedProps: any
    const wrapper = mount(UnifiedDateRange, {
      props: {
        type: 'datetimerange',
        format: 'YYYY-MM-DD HH:mm:ss',
        valueFormat: 'YYYY-MM-DD HH:mm:ss',
      },
      global: {
        stubs: {
          ElDatePicker: {
            props: ['type', 'shortcuts', 'defaultTime'],
            emits: ['calendar-change'],
            setup(props, { emit }) {
              receivedProps = props
              return () => h('button', {
                class: 'stub-picker',
                onClick: () => emit('calendar-change', [new Date(2026, 0, 1), null]),
              })
            },
          },
        },
      },
    })

    expect(wrapper.find('.stub-picker').exists()).toBe(true)
    expect(receivedProps.type).toBe('daterange')
    expect(receivedProps.shortcuts).toHaveLength(1)
    expect(receivedProps.shortcuts[0].text).toBe('先选开始日期')
    expect(receivedProps.shortcuts[0].value()).toBeUndefined()
    expect(receivedProps.defaultTime[0].getHours()).toBe(0)
    expect(receivedProps.defaultTime[1].getHours()).toBe(23)

    await wrapper.find('.stub-picker').trigger('click')
    expect(receivedProps.shortcuts.map((shortcut: any) => shortcut.text)).toEqual([
      '结束到今天',
      '结束到昨天',
      '结束到上月底',
    ])

    const todayRange = receivedProps.shortcuts[0].value()
    expect(todayRange[0]).toEqual(new Date(2026, 0, 1))
    expect(todayRange[1].toDateString()).toBe(new Date().toDateString())

    const yesterday = new Date()
    yesterday.setDate(yesterday.getDate() - 1)
    expect(receivedProps.shortcuts[1].value()[1].toDateString()).toBe(yesterday.toDateString())

    const previousMonthEnd = new Date(new Date().getFullYear(), new Date().getMonth(), 0)
    expect(receivedProps.shortcuts[2].value()[1].toDateString()).toBe(previousMonthEnd.toDateString())
  })

  it('does not add a day shortcut to month ranges', () => {
    let receivedProps: any
    const wrapper = mount(UnifiedDateRange, {
      props: { type: 'monthrange' },
      global: {
        stubs: {
          ElDatePicker: {
            props: ['type', 'shortcuts'],
            setup(props) {
              receivedProps = props
              return () => h('div', { class: 'stub-picker' })
            },
          },
        },
      },
    })

    expect(wrapper.find('.stub-picker').exists()).toBe(true)
    expect(receivedProps.type).toBe('monthrange')
    expect(receivedProps.shortcuts).toEqual([])
  })

  it('formats inbound datetimerange model values to the configured display value format', async () => {
    const wrapper = mount(UnifiedDateRange, {
      props: {
        modelValue: ['2026-04-01T12:30:00Z', '2026-04-10T08:45:15Z'],
        type: 'datetimerange',
        format: 'YYYY-MM-DD HH:mm:ss',
        valueFormat: 'YYYY-MM-DDTHH:mm:ss.SSSZ',
      },
      global: {
        stubs: {
          ElDatePicker: {
            props: ['modelValue'],
            template: '<div class="stub-picker">{{ Array.isArray(modelValue) ? modelValue.join("|") : "" }}</div>',
          },
        },
      },
    })

    expect(wrapper.find('.stub-picker').text()).toBe('2026-04-01T12:30:00.000Z|2026-04-10T08:45:15.000Z')
  })

  it('emits one normalized update when the picker updates the value', async () => {
    const Host = defineComponent({
      components: { UnifiedDateRange },
      setup() {
        const value = ref<[string, string] | null>(null)
        return { value }
      },
      template: `
        <UnifiedDateRange
          v-model="value"
          type="datetimerange"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
        />
      `,
    })

    const wrapper = mount(Host, {
      global: {
        stubs: {
          ElDatePicker: {
            props: ['modelValue'],
            emits: ['update:model-value', 'change'],
            setup(_props, { emit }) {
              return () =>
                h('button', {
                  class: 'stub-trigger',
                  onClick: () => {
                    emit('update:model-value', ['2026-04-01 12:30:45', '2026-04-10 08:45:15'])
                    emit('change', ['2026-04-01 12:30:45', '2026-04-10 08:45:15'])
                  },
                })
            },
          },
        },
      },
    })

    await wrapper.find('.stub-trigger').trigger('click')

    expect(wrapper.vm.value).toEqual(['2026-04-01 12:30:45', '2026-04-10 08:45:15'])
  })

  it('keeps daterange selections visible after the picker updates the model value', async () => {
    const wrapper = mount(UnifiedDateRange, {
      props: {
        modelValue: ['2026-02-01 00:00:00', '2026-02-28 23:59:59'],
        type: 'daterange',
        format: 'YYYY-MM-DD',
        valueFormat: 'YYYY-MM-DD HH:mm:ss',
      },
      global: {
        stubs: {
          ElDatePicker: {
            props: ['modelValue'],
            template: '<div class="stub-picker">{{ Array.isArray(modelValue) ? modelValue.join("|") : "" }}</div>',
          },
        },
      },
    })

    expect(wrapper.find('.stub-picker').text()).toBe('2026-02-01 00:00:00|2026-02-28 23:59:59')
  })
})
