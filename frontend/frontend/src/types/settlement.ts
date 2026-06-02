// 结算任务状态类型
export type TaskStatus = 'pending' | 'running' | 'success' | 'failed';

// 结算配置接口
export interface SettlementConfig {
  id: number;
  daily_time: string; // 每日结算时间，格式: "HH:MM"
  weekly_day: number; // 每周结算日，1-7 代表周一到周日
  weekly_time: string; // 每周结算时间，格式: "HH:MM"
  enabled: boolean; // 是否启用自动结算
  daily_enabled?: boolean; // 是否启用每日自动结算
  weekly_enabled?: boolean; // 是否启用每周自动结算
  node_daily_enabled?: boolean; // 是否启用EDC节点每日自动结算
  node_daily_time?: string; // EDC节点每日结算时间
  node_monthly_enabled?: boolean; // 是否启用EDC节点每月自动结算
  node_monthly_day?: number; // EDC节点月结算触发日
  node_monthly_time?: string; // EDC节点月结算时间
  recalc_after_daily?: boolean; // 日结算完成后自动复算
  recalc_after_weekly?: boolean; // 周结算完成后自动复算
  last_execute_time: string; // 上次执行时间
  update_time: string; // 更新时间
}

// 结算任务接口
export interface SettlementTask {
  id: number;
  task_type: 'daily' | 'weekly' | 'node_daily95' | 'node_monthly95' | 'customer_init' | 'customer_recalc'; // 任务类型
  task_date: string; // 任务日期
  status: TaskStatus; // 任务状态
  start_time: string; // 开始时间
  end_time: string; // 结束时间
  processed_count: number; // 处理的记录数
  total_count?: number; // 预计总量（用于进度/ETA）
  error_message: string; // 错误信息
  create_time: string; // 创建时间
  update_time: string; // 更新时间
}

// 结算任务列表响应
export interface TaskListResponse {
  items: SettlementTask[];
  total: number;
}

// 结算数据接口
export interface Settlement {
  id: number;
  school_id: string; // 学校ID
  school_name: string; // 学校名称
  region: string; // 地区
  cp: string; // 运营商
  service_date?: string; // 服务日期/月份（按月聚合时为 YYYY-MM）
  date: string; // 兼容字段：结算日期
  settlement_value?: number; // 日95原始值（后端 settlement_value）
  daily_95_value: number; // 兼容字段：日95值
  weekly_95_value: number; // 周95值
  monthly_95_value: number; // 月95值
  daily_increment_value?: number; // 当日增量原始值
  create_time: string; // 创建时间
  update_time: string; // 更新时间
}

// 结算数据列表响应
export interface SettlementListResponse {
  items: Settlement[];
  total: number;
}

// 结算结果条目
export interface SettlementResultItem {
  region: string;
  cp: string;
  school_id: string;
  school_name: string;
  billing_days: number;
  average_95_flow: number;
  total_95_flow: number;
  missing_days: number;
  formula_id: number;
  formula_name: string;
  formula_tokens: string;
  customer_fee: number;
  network_line_fee: number;
  node_deduction_fee: number;
  final_fee: number;
  amount: number;
  currency: string;
  start_date: string;
  end_date: string;
  updated_at: string;
  missing_fields: string[];
  calculation_detail: string;
}

export interface SettlementResultResponse {
  items: SettlementResultItem[];
  total: number;
}

export interface SettlementResultFilter {
  region?: string;
  cp?: string;
  school_id?: string;
  school_name?: string;
  start_date: string;
  end_date: string;
  limit?: number;
  offset?: number;
  formula_id?: number;
  unit_base?: number; // 1000=SI(GB), 1024=IEC(GiB)
}

// 渠道维度结算结果条目
export interface ChannelSettlementResultItem {
  user_id: number;
  user_name: string;
  amount: number;
  currency: string;
  start_date: string;
  end_date: string;
  formula_id: number;
  formula_name: string;
  // 聚合分项明细（JSON 字符串，tooltip 展示）
  breakdown_detail?: string; // e.g. { "customer_fee": 123, "network_line_fee": 45, "node_deduction_fee": 6, "final_fee": 72 }
  updated_at?: string;
}

export interface ChannelSettlementResultResponse {
  items: ChannelSettlementResultItem[];
  total: number;
}

export interface ChannelSettlementResultFilter {
  channel_name?: string;
  user_name?: string;
  start_date: string;
  end_date: string;
  limit?: number;
  offset?: number;
  formula_id?: number;
}

// 结算数据筛选条件
export interface SettlementFilter {
  school_id?: string;
  region?: string;
  cp?: string;
  start_date?: string;
  end_date?: string;
  page?: number;
  page_size?: number;
}
