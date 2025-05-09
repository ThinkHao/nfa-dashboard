import axios from 'axios'

// 创建axios实例
const api = axios.create({
  baseURL: 'http://localhost:8081', // 后端API地址
  timeout: 10000 // 请求超时时间
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    // 可以在这里添加请求头等信息
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    // 只返回响应的data部分
    return response.data
  },
  (error) => {
    return Promise.reject(error)
  }
)

// API接口
export default {
  // 获取学校列表
  getSchools(params?: any) {
    return api.get('/api/v1/schools', { params })
  },
  
  // 获取地区列表
  getRegions() {
    return api.get('/api/v1/regions')
  },
  
  // 获取运营商列表
  getCPs() {
    return api.get('/api/v1/cps')
  },
  
  // 获取流量数据
  getTrafficData(params?: any) {
    return api.get('/api/v1/traffic', { params })
  },
  
  // 获取流量汇总数据
  getTrafficSummary(params?: any) {
    return api.get('/api/v1/traffic/summary', { params })
  }
}
