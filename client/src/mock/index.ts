/**
 * Mock数据入口
 * 
 * 根据环境变量VITE_USE_MOCK启用/禁用Mock数据
 * @reference notes/client/1st/Client-Base-Design.md 第6节
 */

import MockAdapter from 'axios-mock-adapter'
import httpClient from '@/utils/http-client'
import { setupSettingsMock } from './settings'
import { setupTaskMock } from './task'

/**
 * 启用Mock数据
 */
export const setupMock = (): void => {
  // 创建Mock适配器（延迟500ms模拟网络延迟）
  const mock = new MockAdapter(httpClient, { delayResponse: 500 })

  // 设置各模块的Mock
  setupSettingsMock(mock)
  setupTaskMock(mock)

  console.log('[Mock] Mock数据已启用')
  console.log('[Mock] 使用VITE_USE_MOCK环境变量控制Mock开关')
}

