import { describe, expect, it, vi } from 'vitest'
import { runSettlementQueryInitialLoads } from '@/views/settlement-query-initial-loads'

describe('runSettlementQueryInitialLoads', () => {
  it('starts every independent loader before waiting for any result', async () => {
    const releases: Array<() => void> = []
    const makeLoader = () => vi.fn(() => new Promise<void>((resolve) => releases.push(resolve)))
    const loaders = [makeLoader(), makeLoader(), makeLoader()]

    const pending = runSettlementQueryInitialLoads(loaders)

    expect(loaders.map((load) => load.mock.calls.length)).toEqual([1, 1, 1])
    expect(releases).toHaveLength(3)
    releases.forEach((release) => release())
    await pending
  })
})
