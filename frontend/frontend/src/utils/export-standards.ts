export const EXPORT_HEADERS = {
  singleUserDetail: ['学校名称', '地区', 'CP', '服务日期', '日95流量值(Mbps)', '客户金额', '线路金额', '节点金额', '渠道金额', '总归属金额'],
  singleUserMonthlyColumnPrefix: ['学校', '存量起算时间', '增量起算时间', '日95均值(Mbps)'],
  daily95Detail: ['日期', '学校名称', '地区', 'CP', '95值(Mbps)'],
  operationLogs: ['时间', '用户ID', '方法', '路径', '状态码', '成功', '耗时(ms)', 'IP', '错误信息'],
  finalCustomerRates: ['区域', 'CP', '学校', '服务日期', '客户费', '客户费(折后)', '线路费', '渠道费率', '客户费归属', '线路费归属', '渠道费归属'],
  customerRatesImportErrors: ['行号', '错误信息'],
} as const

export const EXPORT_FILENAME_PREFIX = {
  settlementDataDetail: '结算数据明细',
  settlementDataMonthlyAgg: '结算数据明细_按月聚合',
  singleUserDaily: '单用户结算_按日明细',
  singleUserMonthly: '单用户结算_按月明细',
  singleUserMonthlyColumn: '单用户结算_按月列视图',
  daily95Detail: '日95明细',
  operationLogs: '操作日志',
  customerRates: '客户业务费率',
  customerRatesTemplate: '客户业务费率_导入模板',
  customerRatesImportErrors: '客户业务费率_导入错误明细',
  finalCustomerRates: '最终客户费率',
} as const
