import { defineStore } from 'pinia'

export type ThemeSize = 'default' | 'small' | 'large'
export interface ThemeState {
  isDark: boolean
  primary: string
  size: ThemeSize
  density: 'normal' | 'compact'
}

const THEME_KEY = 'app_theme_v1'

function applyThemeToDocument(state: ThemeState) {
  try {
    if (state.isDark) {
      document.documentElement.setAttribute('data-theme', 'dark')
      document.documentElement.classList.add('contrast-max')
    } else {
      document.documentElement.removeAttribute('data-theme')
      document.documentElement.classList.remove('contrast-max')
    }
    document.documentElement.style.setProperty('--el-color-primary', state.primary)
    document.documentElement.style.setProperty('--color-primary', state.primary)
    document.documentElement.style.setProperty('--brand-primary', state.primary)
    // 密度可在需要时作用到 body class 或 CSS 变量
    document.documentElement.style.setProperty('--app-density', state.density)
    // 设置 data-density 方便 CSS 选择器做差异化
    document.documentElement.setAttribute('data-density', state.density)
  } catch {}
}

export const useThemeStore = defineStore('theme', {
  state: (): ThemeState => ({
    isDark: false,
    primary: '#1890ff',
    size: 'default',
    density: 'compact',
  }),
  actions: {
    init() {
      try {
        const raw = localStorage.getItem(THEME_KEY)
        if (raw) {
          const cfg = JSON.parse(raw)
          this.isDark = !!cfg.isDark
          this.primary = cfg.primary || this.primary
          this.size = (cfg.size || this.size) as ThemeSize
          // 密度迁移：未设置或仍为 normal，则使用新的默认 compact
          const savedDensity = cfg.density as ThemeState['density'] | undefined
          if (!savedDensity || savedDensity === 'normal') {
            this.density = 'compact'
          } else {
            this.density = savedDensity
          }
        }
      } catch {}
      applyThemeToDocument(this.$state)
      // 持久化一次以写入可能的密度迁移
      this.persist()
    },
    persist() {
      try { localStorage.setItem(THEME_KEY, JSON.stringify(this.$state)) } catch {}
    },
    setDark(v: boolean) {
      this.isDark = v
      this.persist()
      applyThemeToDocument(this.$state)
    },
    toggleDark() {
      this.setDark(!this.isDark)
    },
    setPrimary(color: string) {
      this.primary = color
      this.persist()
      applyThemeToDocument(this.$state)
    },
    setSize(size: ThemeSize) {
      this.size = size
      this.persist()
      // ConfigProvider 将读取 size，无需额外操作
    },
    setDensity(d: ThemeState['density']) {
      this.density = d
      this.persist()
      applyThemeToDocument(this.$state)
    },
  },
})
