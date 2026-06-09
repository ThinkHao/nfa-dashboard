import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import NodeRatesView from '../NodeRatesView.vue'

const apiMock = vi.hoisted(() => ({
  settlementRates: {
    node: {
      list: vi.fn(),
      upsert: vi.fn(),
    },
  },
  system: {
    users: {
      list: vi.fn(),
    },
  },
  v2: {
    edc: {
      getRegions: vi.fn(),
      getCPs: vi.fn(),
      getEntities: vi.fn(),
    },
  },
}))

vi.mock('@/api', () => ({ default: apiMock }))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    hasPermission: () => true,
  }),
}))
vi.mock('@/composables/usePageRefresh', () => ({
  usePageRefresh: vi.fn(),
}))
vi.mock('element-plus', () => ({
  ElMessage: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const inertStub = defineComponent({
  setup() {
    return () => h('div')
  },
})

function mountComponent() {
  return mount(NodeRatesView, {
    global: {
      stubs: {
        ElCard: passthroughStub,
        ElForm: passthroughStub,
        ElFormItem: passthroughStub,
        ElSelect: passthroughStub,
        ElOption: passthroughStub,
        ElButton: passthroughStub,
        ElTable: passthroughStub,
        ElTableColumn: inertStub,
        ElTag: passthroughStub,
        ElTooltip: passthroughStub,
        ElDrawer: passthroughStub,
        ElInput: passthroughStub,
        ElInputNumber: passthroughStub,
        ElSwitch: passthroughStub,
        ElPagination: passthroughStub,
        SearchSelect: passthroughStub,
        QueryActionButton: passthroughStub,
      },
      directives: {
        loading: {},
      },
    },
  })
}

describe('NodeRatesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.v2.edc.getRegions.mockResolvedValue([])
    apiMock.v2.edc.getCPs.mockResolvedValue([])
    apiMock.v2.edc.getEntities.mockResolvedValue({ items: [] })
    apiMock.system.users.list.mockImplementation((params: any) => {
      if (params?.ids === '7') {
        return Promise.resolve({
          items: [{ id: 7, username: 'zhangsan', alias: '张三', display_name: '张三' }],
          total: 1,
        })
      }
      return Promise.resolve({ items: [], total: 0 })
    })
    apiMock.settlementRates.node.list.mockResolvedValue({
      items: [{
        id: 1,
        entity_id: 12,
        display_name: 'BJ-Bilibili',
        region: 'BJ',
        cp: 'Bilibili',
        settlement_mode: 'daily_95_avg',
        settlement_type: 'daily_95_avg',
        unit_base: 1000,
        node_construction_fee: 1.2,
        node_construction_fee_owner_id: 7,
        updated_at: '2026-06-09 10:00:00',
      }],
      total: 1,
    })
  })

  it('preloads current owner labels when opening edit drawer', async () => {
    const wrapper = mountComponent()
    await flushPromises()
    await flushPromises()

    const row = (wrapper.vm as any).displayItems[0]
    ;(wrapper.vm as any).openDialog(row)
    await flushPromises()

    expect((wrapper.vm as any).ownerUserOptions).toEqual(
      expect.arrayContaining([{ id: 7, label: '张三' }]),
    )
  })
})
