export interface QuickAccessItem {
  key: string
  title: string
  description: string
  path: string
  icon: string
  required: string[]
}

const QUICK_ACCESS_REGISTRY: QuickAccessItem[] = [
  {
    key: 'traffic',
    title: '流速监控',
    description: '查看学校流速走势与筛选明细',
    path: '/traffic',
    icon: 'TrendCharts',
    required: ['traffic.read'],
  },
  {
    key: 'traffic-volume',
    title: '流量监控',
    description: '查看院校自然日服务流量',
    path: '/traffic-volume',
    icon: 'Histogram',
    required: ['traffic.read'],
  },
  {
    key: 'settlement',
    title: '结算系统配置',
    description: '执行结算任务并维护结算参数',
    path: '/settlement',
    icon: 'Wallet',
    required: ['settlement.read', 'settlement.calculate'],
  },
  {
    key: 'settlement-rates',
    title: '客户业务费率',
    description: '维护客户/渠道费率与归属配置',
    path: '/settlement/rates/customer',
    icon: 'User',
    required: ['rates.customer.read'],
  },
  {
    key: 'settlement-user-query',
    title: '院校结算查询',
    description: '按用户与时间快速核对结算数据',
    path: '/settlement/user-query',
    icon: 'DataAnalysis',
    required: ['settlement.data.read'],
  },
  {
    key: 'operation-logs',
    title: '操作日志',
    description: '查看最近操作记录与审计轨迹',
    path: '/operation-logs',
    icon: 'Notebook',
    required: ['operation_logs.read'],
  },
]

export function buildQuickAccessItems(permissions: string[]): QuickAccessItem[] {
  const set = new Set(permissions)
  return QUICK_ACCESS_REGISTRY.filter((item) => item.required.some((code) => set.has(code)))
}
