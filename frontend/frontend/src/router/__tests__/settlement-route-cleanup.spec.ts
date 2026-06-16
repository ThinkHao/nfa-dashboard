import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('settlement route cleanup', () => {
  it('removes deprecated settlement pages and their legacy redirect aliases', () => {
    const routes = router.getRoutes()

    expect(routes.find((r) => r.name === 'settlement-dashboard')).toBeUndefined()
    expect(routes.find((r) => r.name === 'settlement-channels')).toBeUndefined()
    expect(routes.find((r) => r.name === 'settlement-entities')).toBeUndefined()

    // 旧的 /settlement/dashboard、/settlement/channels 跳转别名已下线，
    // 未匹配路径由 catch-all 统一回落到首页。
    expect(routes.find((r) => r.path === '/settlement/dashboard')).toBeUndefined()
    expect(routes.find((r) => r.path === '/settlement/channels')).toBeUndefined()
  })

  it('keeps single-user settlement query route unchanged', () => {
    const route = router.getRoutes().find((r) => r.name === 'settlement-user-query')
    expect(route).toBeTruthy()
    expect(route?.path).toBe('/settlement/user-query')
  })
})
