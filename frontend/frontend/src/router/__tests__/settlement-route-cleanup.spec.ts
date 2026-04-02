import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('settlement route cleanup', () => {
  it('removes deprecated settlement pages and keeps redirect aliases', () => {
    const routes = router.getRoutes()

    expect(routes.find((r) => r.name === 'settlement-dashboard')).toBeUndefined()
    expect(routes.find((r) => r.name === 'settlement-channels')).toBeUndefined()
    expect(routes.find((r) => r.name === 'settlement-entities')).toBeUndefined()

    const dashboardAlias = routes.find((r) => r.path === '/settlement/dashboard')
    const channelsAlias = routes.find((r) => r.path === '/settlement/channels')

    expect(dashboardAlias?.redirect).toBe('/settlement/user-query')
    expect(channelsAlias?.redirect).toBe('/settlement/user-query')
  })

  it('keeps single-user settlement query route unchanged', () => {
    const route = router.getRoutes().find((r) => r.name === 'settlement-user-query')
    expect(route).toBeTruthy()
    expect(route?.path).toBe('/settlement/user-query')
  })
})
