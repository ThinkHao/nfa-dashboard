import { flushPromises, mount } from '@vue/test-utils'
import { KeepAlive, defineComponent, nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EDCNFAComparisonView from '../EDCNFAComparisonView.vue'

const mocks = vi.hoisted(() => ({
  getComparisonGroups: vi.fn(),
  getComparison: vi.fn(),
}))

vi.mock('../../api', () => ({
  default: {
    v2: {
      edcNfa: {
        getComparisonGroups: mocks.getComparisonGroups,
        getComparison: mocks.getComparison,
      },
    },
  },
}))

vi.mock('echarts/core', () => ({ use: vi.fn() }))
vi.mock('echarts/renderers', () => ({ CanvasRenderer: {} }))
vi.mock('echarts/charts', () => ({ LineChart: {} }))
vi.mock('echarts/components', () => ({
  GridComponent: {},
  LegendComponent: {},
  TitleComponent: {},
  TooltipComponent: {},
}))
vi.mock('vue-echarts', () => ({ default: { template: '<div />' } }))
vi.mock('element-plus', () => {
  const container = { template: '<div><slot /></div>' }
  return {
    ElCard: container,
    ElMessage: { error: vi.fn() },
    ElOption: { props: ['label'], template: '<span>{{ label }}</span>' },
    ElPagination: container,
    ElSelect: container,
    ElTable: container,
    ElTableColumn: { template: '<div><slot :row="{}" /></div>' },
  }
})
vi.mock('../../components/ui/QueryActionButton.vue', () => ({ default: { template: '<button />' } }))
vi.mock('../../components/ui/UnifiedDateRange.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../../composables/useCancelableQuery', () => ({
  useCancelableQuery: () => ({
    running: ref(false),
    run: async (query: (signal: AbortSignal) => Promise<void>) => query(new AbortController().signal),
  }),
}))

describe('EDCNFAComparisonView mapping options', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getComparisonGroups
      .mockResolvedValueOnce([{ id: 1, group_name: 'BJ-Bilibili', nfa_src_region: '北京市', nfa_cp: 'bilibili' }])
      .mockResolvedValueOnce([
        { id: 1, group_name: 'BJ-Bilibili', nfa_src_region: '北京市', nfa_cp: 'bilibili' },
        { id: 2, group_name: 'GD-Bilibili', nfa_src_region: '广东省', nfa_cp: 'bilibili' },
      ])
    mocks.getComparison.mockResolvedValue([])
  })

  it('reloads mapping options when returning to the cached comparison page', async () => {
    const active = ref(true)
    const Host = defineComponent({
      components: { KeepAlive, EDCNFAComparisonView },
      setup: () => ({ active }),
      template: '<KeepAlive><EDCNFAComparisonView v-if="active" /></KeepAlive>',
    })
    const wrapper = mount(Host, { global: { directives: { loading: {} } } })

    await flushPromises()
    expect(wrapper.text()).not.toContain('GD-Bilibili')

    active.value = false
    await nextTick()
    active.value = true
    await nextTick()
    await flushPromises()

    expect(mocks.getComparisonGroups).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('GD-Bilibili（广东省 / bilibili）')
    wrapper.unmount()
  })
})
