import './assets/main.css'
import './assets/theme.css'
import './styles/variables.css'
import './styles/dark.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs' // 导入中文语言包

import App from './App.vue'
import router from './router'
import { useThemeStore } from '@/stores/theme'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(ElementPlus, {
  locale: zhCn, // 使用中文语言包
  size: 'default'
})

// 初始化主题（从 localStorage 读取并应用到文档与 Element Plus）
const themeStore = useThemeStore(pinia)
themeStore.init()

// 全局错误处理
app.config.errorHandler = (err, instance, info) => {
  console.error('全局错误:', err)
  console.error('错误信息:', info)
}


app.mount('#app')
