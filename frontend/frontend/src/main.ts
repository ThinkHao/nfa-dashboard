import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs' // 导入中文语言包

import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus, {
  locale: zhCn, // 使用中文语言包
  size: 'default'
})

// 全局错误处理
app.config.errorHandler = (err, instance, info) => {
  console.error('全局错误:', err)
  console.error('错误信息:', info)
}

app.mount('#app')
