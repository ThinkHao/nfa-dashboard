import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, inject, provide, ref } from 'vue'

import SettlementDataTab from '../SettlementDataTab.vue'

const apiMock = vi.hoisted(() => ({
  v2: {
    getRegions: vi.fn(),
    getCPs: vi.fn(),
    getSchools: vi.fn(),
  },
  settlementData: {
    list: vi.fn(),
    monthlyList: vi.fn(),
    nodeList: vi.fn(),
    nodeMonthlyList: vi.fn(),
    ownerSubjects: vi.fn(),
  },
  settlementRates: {
    customer: {
      list: vi.fn(),
    },
    discountRules: {
      get: vi.fn(),
    },
  },
  system: {
    users: {
      list: vi.fn(),
    },
  },
}))

vi.mock('@/api', () => ({ default: apiMock }))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    hasPermission: () => false,
  }),
}))
vi.mock('@/stores/tasks', () => ({
  useTasksStore: () => ({
    start: vi.fn(),
    update: vi.fn(),
    success: vi.fn(),
    fail: vi.fn(),
  }),
}))
vi.mock('@/composables/usePageRefresh', () => ({
  usePageRefresh: vi.fn(),
}))
vi.mock('@/composables/useSystemTrafficSettings', () => ({
  useSystemTrafficSettings: () => ({
    settings: ref({
      settlement_data_rate_unit: 'Mbps',
      traffic_byte_unit_base: 1000,
      settlement_result_unit_base: 1000,
      settlement_daily_detail_rate_unit: 'Mbps',
      settlement_single_user_rate_unit: 'Gbps',
      hide_non_settlement_schools_in_traffic: false,
    }),
    ensureLoaded: vi.fn().mockResolvedValue(undefined),
  }),
}))

const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const tableStub = defineComponent({
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    provide('tableProps', props)
    return () => h('div', (props.data as any[]).map((row) => h('div', slots.default?.({ row }))))
  },
})
const tableColumnStub = defineComponent({
  setup(_, { slots }) {
    const tableProps = inject<{ data: any[] }>('tableProps', { data: [] })
    return () => h('div', (tableProps.data || []).flatMap((row) => slots.default?.({ row }) || []))
  },
})
const tooltipStub = defineComponent({
  name: 'ElTooltip',
  emits: ['show'],
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})
const inertStub = defineComponent({
  setup() {
    return () => h('div')
  },
})

function mountComponent() {
  return mount(SettlementDataTab, {
    global: {
      stubs: {
        ElCard: passthroughStub,
        ElForm: passthroughStub,
        ElFormItem: passthroughStub,
        ElSegmented: passthroughStub,
        ElSelect: passthroughStub,
        ElOption: passthroughStub,
        ElInput: passthroughStub,
        ElButton: passthroughStub,
        ElTag: passthroughStub,
        ElDialog: passthroughStub,
        ElCheckboxGroup: passthroughStub,
        ElCheckbox: passthroughStub,
        ElDivider: passthroughStub,
        ElPagination: passthroughStub,
        ElTooltip: tooltipStub,
        ElTable: tableStub,
        ElTableColumn: tableColumnStub,
        UnifiedDateRange: passthroughStub,
        SearchSelect: passthroughStub,
        QueryActionButton: passthroughStub,
      },
      directives: {
        loading: {},
      },
    },
  })
}

describe('SettlementDataTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.v2.getRegions.mockResolvedValue([])
    apiMock.v2.getCPs.mockResolvedValue([])
    apiMock.v2.getSchools.mockResolvedValue({ items: [] })
    apiMock.settlementData.ownerSubjects.mockResolvedValue([])
    apiMock.system.users.list.mockResolvedValue({ items: [] })
    apiMock.settlementRates.customer.list.mockResolvedValue({ items: [] })
    apiMock.settlementRates.discountRules.get.mockResolvedValue({ rule: {}, items: [] })
    apiMock.settlementData.list.mockResolvedValue({
      items: [
        {
          region: '华北',
          cp: 'CT',
          school_name: '学校A',
          service_date: '2026-05-01',
          settlement_value: 75000000,
          customer_bill: 12.34,
          customer_fee_owner_id: 1,
          discount_rule_id: 9,
        },
      ],
      total: 1,
    })
  })

  it('does not preload rate and discount metadata when loading the NFA list', async () => {
    mountComponent()

    await flushPromises()
    await flushPromises()

    expect(apiMock.settlementData.list).toHaveBeenCalledTimes(1)
    expect(apiMock.settlementRates.customer.list).not.toHaveBeenCalled()
    expect(apiMock.settlementRates.discountRules.get).not.toHaveBeenCalled()
  })

  it('loads rate and discount metadata lazily once when an amount tooltip shows', async () => {
    const wrapper = mountComponent()

    await flushPromises()
    await flushPromises()

    const tooltip = wrapper.findAllComponents(tooltipStub)[0]
    expect(tooltip).toBeTruthy()

    tooltip.vm.$emit('show')
    await flushPromises()
    tooltip.vm.$emit('show')
    await flushPromises()

    expect(apiMock.settlementRates.discountRules.get).toHaveBeenCalledTimes(1)
    expect(apiMock.settlementRates.discountRules.get).toHaveBeenCalledWith(9)
    expect(apiMock.settlementRates.customer.list).toHaveBeenCalledTimes(1)
    expect(apiMock.settlementRates.customer.list).toHaveBeenCalledWith({
      region: '华北',
      cp: 'CT',
      school_name: '学校A',
      page: 1,
      page_size: 200,
    })
  })
})
