// API响应通用类型定义

// 通用API响应接口
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

// 分页数据接口
export interface PaginatedData<T = any> {
  items: T[];
  total: number;
}

// 分页请求参数
export interface PaginationParams {
  page?: number;
  page_size?: number;
  limit?: number;
  offset?: number;
}

// 学校信息接口
export interface School {
  school_id: string;
  school_name: string;
  region?: string;
  cp?: string;
  create_time?: string;
  update_time?: string;
}

// 流量数据接口
export interface TrafficData {
  id?: number;
  school_id: string;
  create_time: string;
  total_recv: number;
  total_send: number;
  [key: string]: any;
}
