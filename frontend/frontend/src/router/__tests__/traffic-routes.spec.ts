import { describe, expect, it } from 'vitest'

import router from '@/router'

describe('traffic routes', () => {
  it('keeps the speed page and adds a separate daily volume page', () => {
    const routes = router.getRoutes()
    const speed = routes.find((route) => route.name === 'traffic')
    const volume = routes.find((route) => route.name === 'traffic-volume')
    const comparison = routes.find((route) => route.name === 'traffic-comparison')
    const mappings = routes.find((route) => route.name === 'edc-nfa-mappings')

    expect(speed?.path).toBe('/traffic')
    expect(speed?.meta.title).toBe('流速监控')
    expect(volume?.path).toBe('/traffic-volume')
    expect(volume?.meta.title).toBe('流量监控')
    expect(volume?.meta.permissions).toEqual(['traffic.read'])
    expect(comparison?.path).toBe('/traffic-comparison')
    expect(comparison?.meta.permissions).toEqual(['traffic.read'])
    expect(comparison?.meta.group).toBe('traffic-analysis')
    expect(mappings?.path).toBe('/edc-nfa-mappings')
    expect(mappings?.meta.group).toBe('traffic-analysis')
  })
})
