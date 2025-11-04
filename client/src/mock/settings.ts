/**
 * 配置管理API Mock实现
 * 
 * @reference notes/client/2nd/API-Types.md 第6.1节
 */

import type MockAdapter from 'axios-mock-adapter'
import type { GetSettingsResponse, UpdateSettingsRequest, UpdateSettingsResponse } from '@/api/types'

// Mock配置数据（模拟后端Redis存储）
const mockSettingsData: GetSettingsResponse = {
  version: 1,
  is_configured: false, // 初始未配置
  processing_mode: 'standard',

  asr_provider: '',
  asr_api_key: '',
  asr_endpoint: '',

  audio_separation_enabled: false,

  polishing_enabled: false,
  polishing_provider: '',
  polishing_api_key: '',
  polishing_custom_prompt: '',
  polishing_video_type: '',

  translation_provider: '',
  translation_api_key: '',
  translation_endpoint: '',
  translation_video_type: '',

  optimization_enabled: false,
  optimization_provider: '',
  optimization_api_key: '',

  voice_cloning_provider: '',
  voice_cloning_api_key: '',
  voice_cloning_endpoint: '',
  voice_cloning_auto_select_reference: true,

  s2st_provider: '',
  s2st_api_key: ''
}

export const setupSettingsMock = (mock: MockAdapter): void => {
  // GET /v1/settings
  mock.onGet('/v1/settings').reply(() => {
    console.log('[Mock] GET /v1/settings')
    return [200, mockSettingsData]
  })

  // POST /v1/settings
  mock.onPost('/v1/settings').reply(config => {
    console.log('[Mock] POST /v1/settings')

    const request: UpdateSettingsRequest = JSON.parse(config.data)

    // 模拟乐观锁检查
    if (request.version !== mockSettingsData.version) {
      return [
        409,
        {
          code: 'CONFLICT',
          message: '配置已被其他用户修改，请刷新后重试',
          current_version: mockSettingsData.version
        }
      ]
    }

    // 模拟版本号递增
    const newVersion = mockSettingsData.version + 1

    // 更新Mock数据
    Object.assign(mockSettingsData, request, { version: newVersion })

    // 更新is_configured状态
    const hasRequiredConfig =
      mockSettingsData.asr_api_key &&
      mockSettingsData.translation_api_key &&
      mockSettingsData.voice_cloning_api_key
    mockSettingsData.is_configured = Boolean(hasRequiredConfig)

    const response: UpdateSettingsResponse = {
      version: newVersion,
      message: '配置已成功更新'
    }

    return [200, response]
  })
}

