import { describe, expect, it } from 'vitest'
import { normalizePaginatedResponse } from '@/utils/pagination'

describe('normalizePaginatedResponse', () => {
  it('preserves the server total instead of using the current page length', () => {
    const items = Array.from({ length: 10 }, (_, index) => ({ id: index + 1 }))
    expect(normalizePaginatedResponse({ items, total: 67 })).toEqual({ items, total: 67 })
  })

  it('supports legacy array responses', () => {
    const items = [{ id: 1 }, { id: 2 }]
    expect(normalizePaginatedResponse(items)).toEqual({ items, total: 2 })
  })

  it('falls back to page length for an invalid total', () => {
    const items = [{ id: 1 }]
    expect(normalizePaginatedResponse({ items, total: 'invalid' })).toEqual({ items, total: 1 })
  })

  it('returns an empty page for unsupported responses', () => {
    expect(normalizePaginatedResponse(null)).toEqual({ items: [], total: 0 })
  })
})
