// 结算任务状态类型
export type TaskStatus = 'pending' | 'running' | 'success' | 'failed';

// 结算配置接口
export interface SettlementConfig {
  id: number;
  daily_time: string; // 每日结算时间，格式: "HH:MM"
  weekly_day: number; // 每周结算日，1-7 代表周一到周日
  weekly_time: string; // 每周结算时间，格式: "HH:MM"
  enabled: boolean; // 是否启用自动结算
  last_execute_time: string; // 上次执行时间
  update_time: string; // 更新时间
}

// 结算任务接口
export interface SettlementTask {
  id: number;
  task_type: 'daily' | 'weekly'; // 任务类型：日结算或周结算
  task_date: string; // 任务日期
  status: TaskStatus; // 任务状态
  start_time: string; // 开始时间
  end_time: string; // 结束时间
  processed_count: number; // 处理的记录数
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
  date: string; // 结算日期
  daily_95_value: number; // 日95值
  weekly_95_value: number; // 周95值
  monthly_95_value: number; // 月95值
  create_time: string; // 创建时间
  update_time: string; // 更新时间
}

// 结算数据列表响应
export interface SettlementListResponse {
  items: Settlement[];
  total: number;
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
