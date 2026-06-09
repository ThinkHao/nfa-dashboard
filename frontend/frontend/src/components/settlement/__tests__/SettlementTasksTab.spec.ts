import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import SettlementTasksTab from '../SettlementTasksTab.vue'

const apiMock = vi.hoisted(() => ({
  settlement: {
    getTasks: vi.fn(),
    createDailyTask: vi.fn(),
    createWeeklyTask: vi.fn(),
    createNodeDailyTask: vi.fn(),
    createNodeMonthlyTask: vi.fn(),
    getTaskById: vi.fn(),
    deleteTask: vi.fn(),
  },
  v2: {
    getSchools: vi.fn(),
  },
}))

const cleanupMock = vi.hoisted(() => vi.fn())

vi.mock('@/api', () => ({ default: apiMock }))
vi.mock('../../api', () => ({ default: apiMock }))
vi.mock('@/utils/overlayCleanup', () => ({
  cleanupStaleElementOverlays: cleanupMock,
}))
vi.mock('element-plus', () => ({
  ElMessage: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
  ElMessageBox: {
    confirm: vi.fn(),
  },
}))
vi.mock('@/stores/tasks', () => ({
  useTasksStore: () => ({
    upsertSettlementTask: vi.fn(),
  }),
}))
vi.mock('@/composables/usePageRefresh', () => ({
  usePageRefresh: vi.fn(),
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
  return mount(SettlementTasksTab, {
    global: {
      stubs: {
        ElCard: passthroughStub,
        ElForm: passthroughStub,
        ElFormItem: passthroughStub,
        ElSelect: passthroughStub,
        ElOption: passthroughStub,
        ElButton: passthroughStub,
        ElTag: passthroughStub,
        ElDialog: passthroughStub,
        ElDatePicker: passthroughStub,
        ElTable: passthroughStub,
        ElTableColumn: inertStub,
        ElPagination: passthroughStub,
        UnifiedDateRange: passthroughStub,
        QueryActionButton: passthroughStub,
      },
      directives: {
        loading: {},
      },
    },
  })
}

function deferred<T = unknown>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('SettlementTasksTab create dialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.settlement.getTasks.mockResolvedValue({ items: [], total: 0 })
    apiMock.v2.getSchools.mockResolvedValue({ items: [] })
  })

  it('does not submit node monthly tasks twice while a create request is running', async () => {
    const requests = [deferred(), deferred()]
    apiMock.settlement.createNodeMonthlyTask
      .mockImplementationOnce(() => requests[0].promise)
      .mockImplementationOnce(() => requests[1].promise)
    const wrapper = mountComponent()
    await flushPromises()

    ;(wrapper.vm as any).createNodeMonthlyTask()
    const first = (wrapper.vm as any).submitTaskCreate()
    const second = (wrapper.vm as any).submitTaskCreate()

    expect(apiMock.settlement.createNodeMonthlyTask).toHaveBeenCalledTimes(1)
    requests[0].resolve({})
    requests[1].resolve({})
    await Promise.all([first, second])
  })

  it('submits one node daily create request for a date range', async () => {
    apiMock.settlement.createNodeDailyTask.mockResolvedValue({})
    const wrapper = mountComponent()
    await flushPromises()

    ;(wrapper.vm as any).createNodeDailyTask()
    ;(wrapper.vm as any).taskForm.dateRange = ['2026-06-01', '2026-06-30']
    await (wrapper.vm as any).submitTaskCreate()
    await flushPromises()

    expect(apiMock.settlement.createNodeDailyTask).toHaveBeenCalledTimes(1)
    expect(apiMock.settlement.createNodeDailyTask).toHaveBeenCalledWith({
      start_date: '2026-06-01',
      end_date: '2026-06-30',
    })
  })

  it('submits one node monthly create request for a month range', async () => {
    apiMock.settlement.createNodeMonthlyTask.mockResolvedValue({})
    const wrapper = mountComponent()
    await flushPromises()

    ;(wrapper.vm as any).createNodeMonthlyTask()
    ;(wrapper.vm as any).taskForm.monthRange = ['2026-01', '2026-12']
    await (wrapper.vm as any).submitTaskCreate()
    await flushPromises()

    expect(apiMock.settlement.createNodeMonthlyTask).toHaveBeenCalledTimes(1)
    expect(apiMock.settlement.createNodeMonthlyTask).toHaveBeenCalledWith({
      start_month: '2026-01',
      end_month: '2026-12',
    })
  })

  it('blocks dialog close while create request is running and allows it afterwards', async () => {
    const wrapper = mountComponent()
    await flushPromises()
    const done = vi.fn()

    ;(wrapper.vm as any).submitting = true
    ;(wrapper.vm as any).beforeCloseCreateTask(done)
    expect(done).not.toHaveBeenCalled()

    ;(wrapper.vm as any).submitting = false
    ;(wrapper.vm as any).beforeCloseCreateTask(done)
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('cleans stale overlays after the create dialog closes', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    ;(wrapper.vm as any).onCreateTaskDialogClosed()

    expect(cleanupMock).toHaveBeenCalledWith(document)
  })
})
