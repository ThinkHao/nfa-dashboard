import { ref } from 'vue'
import api from '@/api'
import type { SystemTrafficSettings } from '@/types/api'
import { normalizeByteUnitBase, normalizeRateUnit } from '@/utils/traffic-units'

const defaultSettings: SystemTrafficSettings = {
  hide_non_settlement_schools_in_traffic: false,
  traffic_byte_unit_base: 1024,
  settlement_result_unit_base: 1024,
  settlement_data_rate_unit: 'Mbps',
  settlement_daily_detail_rate_unit: 'Mbps',
  settlement_single_user_rate_unit: 'Gbps',
}

const settingsRef = ref<SystemTrafficSettings>({ ...defaultSettings })
const loadedRef = ref(false)
let loadingPromise: Promise<SystemTrafficSettings> | null = null

function sanitizeSettings(input?: Partial<SystemTrafficSettings> | null): SystemTrafficSettings {
  return {
    hide_non_settlement_schools_in_traffic: !!input?.hide_non_settlement_schools_in_traffic,
    traffic_byte_unit_base: normalizeByteUnitBase(input?.traffic_byte_unit_base, 1024),
    settlement_result_unit_base: normalizeByteUnitBase(input?.settlement_result_unit_base, 1024),
    settlement_data_rate_unit: normalizeRateUnit(input?.settlement_data_rate_unit, 'Mbps'),
    settlement_daily_detail_rate_unit: normalizeRateUnit(input?.settlement_daily_detail_rate_unit, 'Mbps'),
    settlement_single_user_rate_unit: normalizeRateUnit(input?.settlement_single_user_rate_unit, 'Gbps'),
  }
}

export function getSystemTrafficSettingsSnapshot(): SystemTrafficSettings {
  return settingsRef.value
}

export function useSystemTrafficSettings() {
  const apply = (next?: Partial<SystemTrafficSettings> | null): SystemTrafficSettings => {
    const sanitized = sanitizeSettings(next)
    settingsRef.value = sanitized
    loadedRef.value = true
    return sanitized
  }

  const ensureLoaded = async (force = false): Promise<SystemTrafficSettings> => {
    if (!force && loadedRef.value) return settingsRef.value
    if (!force && loadingPromise) return loadingPromise
    loadingPromise = api.system.settings.getTraffic()
      .then((cfg) => apply(cfg))
      .catch(() => {
        if (!loadedRef.value) {
          apply(defaultSettings)
        }
        return settingsRef.value
      })
      .finally(() => {
        loadingPromise = null
      })
    return loadingPromise
  }

  return {
    settings: settingsRef,
    loaded: loadedRef,
    ensureLoaded,
    apply,
    defaults: defaultSettings,
  }
}
