import { get, post, put, del } from '../../utils/request'

// --- Types ---

export interface QueryDataSource {
  id: string
  tenant_id: number
  name: string
  type: string  // mysql, postgresql, sqlite
  config: {
    host?: string
    port?: number
    username?: string
    password?: string
    database?: string
    charset?: string
    file_path?: string
    sslmode?: string
  }
  description?: string
  status: 'active' | 'inactive' | 'error'
  error_message?: string
  created_at: string
  updated_at: string
}

// --- API calls ---

/**
 * 创建查询数据源
 */
export function createQueryDataSource(data: Partial<QueryDataSource>) {
  return post('/api/v1/query-data-sources', data)
}

/**
 * 获取查询数据源列表
 */
export function listQueryDataSources() {
  return get('/api/v1/query-data-sources')
}

/**
 * 获取查询数据源详情
 */
export function getQueryDataSource(id: string) {
  return get(`/api/v1/query-data-sources/${id}`)
}

/**
 * 更新查询数据源
 */
export function updateQueryDataSource(id: string, data: Partial<QueryDataSource>) {
  return put(`/api/v1/query-data-sources/${id}`, data)
}

/**
 * 删除查询数据源
 */
export function deleteQueryDataSource(id: string) {
  return del(`/api/v1/query-data-sources/${id}`)
}

/**
 * 验证数据源连接
 */
export function validateQueryDataSourceConnection(data: Partial<QueryDataSource>) {
  return post('/api/v1/query-data-sources/validate', data)
}
