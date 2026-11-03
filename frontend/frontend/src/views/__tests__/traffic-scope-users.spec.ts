import { describe, expect, it, vi } from 'vitest'

import {
  isTrafficScopeUserListNearBottom,
  useTrafficScopeUsers,
} from '../traffic-scope-users'

describe('traffic-scope-users', () => {
  it('loads one page first and appends the next page on demand', async () => {
    const listUsers = vi.fn(async ({ page }: { page: number; page_size: number; username?: string }) => ({
      items: Array.from({ length: page === 3 ? 1 : 2 }, (_, index) => ({
        id: (page - 1) * 2 + index + 1,
        username: `user-${(page - 1) * 2 + index + 1}`,
        display_name: `User ${(page - 1) * 2 + index + 1}`,
        status: 1,
      })),
      total: 5,
    }))
    const options = useTrafficScopeUsers(listUsers, 2)

    await options.search()

    expect(listUsers).toHaveBeenCalledTimes(1)
    expect(options.users.value.map((user) => user.username)).toEqual(['user-1', 'user-2'])
    expect(options.hasMore.value).toBe(true)

    await options.loadMore()

    expect(listUsers).toHaveBeenNthCalledWith(2, { username: undefined, page: 2, page_size: 2 })
    expect(options.users.value.map((user) => user.username)).toEqual([
      'user-1',
      'user-2',
      'user-3',
      'user-4',
    ])
  })

  it('loads more options when the dropdown scroll reaches its bottom', () => {
    expect(isTrafficScopeUserListNearBottom({ scrollTop: 100, clientHeight: 200, scrollHeight: 400 })).toBe(false)
    expect(isTrafficScopeUserListNearBottom({ scrollTop: 180, clientHeight: 200, scrollHeight: 400 })).toBe(true)
  })

  it('searches by username or alias on the server', async () => {
    const listUsers = vi.fn(async () => ({
      items: [{ id: 7, username: 'huangwl', display_name: 'Huang Wei Lin', status: 1 }],
      total: 1,
    }))
    const options = useTrafficScopeUsers(listUsers, 50)

    await options.search(' huangwl ')

    expect(listUsers).toHaveBeenCalledWith({ username: 'huangwl', page: 1, page_size: 50 })
    expect(options.users.value[0]?.username).toBe('huangwl')
    expect(options.hasMore.value).toBe(false)
  })

  it('ignores a stale result when a newer search finishes first', async () => {
    type UserPage = {
      items: { id: number; username: string; display_name: string; status: number }[]
      total: number
    }
    let resolveOldSearch!: (page: UserPage) => void
    const listUsers = vi.fn(({ username }: { username?: string; page: number; page_size: number }) => {
      if (username === 'old') {
        return new Promise<UserPage>((resolve) => {
          resolveOldSearch = resolve
        })
      }
      return Promise.resolve({
        items: [{ id: 2, username: 'huangwl', display_name: 'Huang Wei Lin', status: 1 }],
        total: 1,
      })
    })
    const options = useTrafficScopeUsers(listUsers, 50)

    const oldSearch = options.search('old')
    await options.search('huangwl')
    resolveOldSearch({
      items: [{ id: 1, username: 'old-user', display_name: 'Old User', status: 1 }],
      total: 1,
    })
    await oldSearch

    expect(options.users.value.map((user) => user.username)).toEqual(['huangwl'])
  })
})
