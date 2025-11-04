/**
 * 配置管理API封装
 * 
 * @reference notes/client/2nd/API-Types.md
 */

import httpClient from '@/utils/http-client'
import type { GetSettingsResponse, UpdateSettingsRequest, UpdateSettingsResponse } from './types'

/**
 * 获取应用配置
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第276-277行
 * @returns GetSettingsResponse
 */
export const getSettings = async (): Promise<GetSettingsResponse> => {
  const response = await httpClient.get<GetSettingsResponse>('/v1/settings')
  return response.data
}

/**
 * 更新应用配置
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第279-281行
 * @param request 更新请求（仅包含需要修改的字段）
 * @returns UpdateSettingsResponse
 * @throws {APIError} 409 Conflict - 配置版本冲突（需要刷新后重试）
 */
export const updateSettings = async (
  request: UpdateSettingsRequest
): Promise<UpdateSettingsResponse> => {
  const response = await httpClient.post<UpdateSettingsResponse>('/v1/settings', request)
  return response.data
}

