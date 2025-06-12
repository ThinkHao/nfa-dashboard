import axios from 'axios'

// 获取当前主机名和协议
const getBaseUrl = () => {
  // 在浏览器环境中使用window.location
  if (typeof window !== 'undefined') {
    const { protocol, hostname } = window.location;
    // 使用与当前页面相同的主机名，但端口使用8081
    return `${protocol}//${hostname}:8081`;
  }
  // 默认值，用于SSR或回退
  return 'http://localhost:8081';
};

// 创建axios实例
const api = axios.create({
  baseURL: getBaseUrl(), // 动态设置后端API地址
  timeout: 60000, // 请求超时时间增加到60秒，以处理大量数据
  maxContentLength: 50 * 1024 * 1024, // 最大内容长度50MB
  maxBodyLength: 50 * 1024 * 1024 // 最大请求体长度50MB
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
  },

  // 结算系统相关API
  settlement: {
    // 获取结算配置
    getConfig() {
      return api.get('/api/v1/settlement/config')
    },

    // 更新结算配置
    updateConfig(config: any) {
      return api.put('/api/v1/settlement/config', config)
    },

    // 获取结算任务列表
    getTasks(params?: any) {
      return api.get('/api/v1/settlement/tasks', { params })
    },

    // 获取结算任务详情
    getTaskById(id: number) {
      return api.get(`/api/v1/settlement/tasks/${id}`)
    },

    // 创建日结算任务
    createDailyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/daily', params)
    },

    // 创建周结算任务
    createWeeklyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/weekly', params)
    },

    // 删除结算任务
    deleteTask(id: number) {
      return api.delete(`/api/v1/settlement/tasks/${id}`)
    },

    // 获取结算数据列表
    getSettlements(params?: any) {
      return api.get('/api/v1/settlement/data', { params })
    },

    // 获取日95明细数据列表
    getDailySettlementDetails(params?: any) {
      return api.get('/api/v1/settlement/daily-details', { params })
    }
  }
}
