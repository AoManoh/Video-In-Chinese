/**
 * 任务管理API Mock实现
 * 
 * @reference notes/client/2nd/API-Types.md 第6.2-6.5节
 */

import type MockAdapter from 'axios-mock-adapter'
import type { UploadTaskResponse, GetTaskStatusResponse } from '@/api/types'

// 模拟任务状态存储
const taskStatusMap = new Map<string, GetTaskStatusResponse>()

export const setupTaskMock = (mock: MockAdapter): void => {
  // POST /v1/tasks/upload
  mock.onPost('/v1/tasks/upload').reply(() => {
    console.log('[Mock] POST /v1/tasks/upload')

    // 生成随机任务ID
    const taskId = crypto.randomUUID()

    const response: UploadTaskResponse = {
      task_id: taskId
    }

    // 初始化任务状态
    taskStatusMap.set(taskId, {
      task_id: taskId,
      status: 'PENDING'
    })

    // 模拟状态变化：3秒后变为PROCESSING
    setTimeout(() => {
      const currentStatus = taskStatusMap.get(taskId)
      if (currentStatus) {
        taskStatusMap.set(taskId, {
          ...currentStatus,
          status: 'PROCESSING'
        })
        console.log(`[Mock] 任务${taskId.slice(0, 8)}状态变更: PROCESSING`)
      }
    }, 3000)

    // 模拟状态变化：10秒后变为COMPLETED
    setTimeout(() => {
      const currentStatus = taskStatusMap.get(taskId)
      if (currentStatus) {
        taskStatusMap.set(taskId, {
          ...currentStatus,
          status: 'COMPLETED',
          result_url: `/v1/tasks/download/${taskId}/result.mp4`
        })
        console.log(`[Mock] 任务${taskId.slice(0, 8)}状态变更: COMPLETED`)
      }
    }, 10000)

    return [200, response]
  })

  // GET /v1/tasks/:taskId/status
  mock.onGet(/\/v1\/tasks\/(.+)\/status/).reply(config => {
    const match = config.url?.match(/\/v1\/tasks\/(.+)\/status/)
    const taskId = match?.[1]

    console.log('[Mock] GET /v1/tasks/:taskId/status', taskId)

    if (!taskId || !taskStatusMap.has(taskId)) {
      return [
        404,
        {
          code: 'NOT_FOUND',
          message: '任务不存在'
        }
      ]
    }

    const status = taskStatusMap.get(taskId)!
    return [200, status]
  })

  // GET /v1/tasks/download/:taskId/:fileName
  mock.onGet(/\/v1\/tasks\/download\/(.+)\/(.+)/).reply(config => {
    const match = config.url?.match(/\/v1\/tasks\/download\/(.+)\/(.+)/)
    const taskId = match?.[1]
    const fileName = match?.[2]

    console.log('[Mock] GET /v1/tasks/download/:taskId/:fileName', taskId, fileName)

    // 返回一个空的Blob（实际应该返回视频文件）
    const mockVideoContent = 'Mock video file content'
    const blob = new Blob([mockVideoContent], { type: 'video/mp4' })

    return [200, blob]
  })
}

