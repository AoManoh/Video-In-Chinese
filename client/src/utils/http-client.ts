/**
 * Axios HTTP客户端封装
 * 
 * 统一配置、请求/响应拦截器、错误处理
 * @reference notes/client/1st/Client-Base-Design.md 第5.3节
 */

import axios, { type AxiosError } from 'axios'
import type { APIError } from '@/api/types'
import { ElMessage } from 'element-plus'

/**
 * 创建Axios实例
 */
const httpClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 300000, // 5分钟超时（对齐后端HTTP_TIMEOUT_SECONDS）
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * 请求拦截器
 */
httpClient.interceptors.request.use(
  config => {
    // 开发环境：打印请求信息
    if (import.meta.env.DEV) {
      console.log('[API Request]', config.method?.toUpperCase(), config.url)
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

/**
 * 响应拦截器
 */
httpClient.interceptors.response.use(
  response => {
    return response
  },
  (error: AxiosError<APIError>) => {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status
      const errorData: any = error.response?.data
      const message = (errorData?.message ?? errorData?.msg ?? '请求失败')

      // 根据后端错误码映射（Gateway-design.md 第8章）
      switch (status) {
        case 400:
          ElMessage.error(`请求参数错误: ${message}`)
          break
        case 404:
          ElMessage.error('资源不存在')
          break
        case 409:
          // 版本冲突（由业务层处理）
          ElMessage.warning('配置冲突，请刷新后重试')
          break
        case 413:
          ElMessage.error('文件太大，超过2048MB限制')
          break
        case 415:
          ElMessage.error('不支持的文件格式')
          break
        case 503:
          ElMessage.error('服务暂时不可用')
          break
        case 507:
          ElMessage.error('服务器存储空间不足')
          break
        default:
          ElMessage.error(message)
      }
    } else {
      ElMessage.error('网络错误')
    }

    return Promise.reject(error)
  }
)

export default httpClient

