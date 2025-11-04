/**
 * 任务轮询器
 * 
 * 实现指数退避策略（3s → 6s → 10s）
 * @reference notes/client/2nd/TaskList-Page-Design.md 第5.1节
 */

import { getTaskStatus } from '@/api/task-api'
import type { GetTaskStatusResponse } from '@/api/types'

type StatusCallback = (status: GetTaskStatusResponse) => void

/**
 * 任务轮询器类
 */
export class TaskPoller {
  private timers: Map<string, number> = new Map()
  private intervals: Map<string, number> = new Map()

  /**
   * 开始轮询单个任务
   * 
   * @param taskId 任务ID
   * @param callback 状态更新回调
   */
  start(taskId: string, callback: StatusCallback) {
    // 初始间隔3秒
    this.intervals.set(taskId, 3000)
    this.poll(taskId, callback)
  }

  /**
   * 执行轮询
   */
  private async poll(taskId: string, callback: StatusCallback): Promise<void> {
    try {
      const response = await getTaskStatus(taskId)
      callback(response)

      // 任务完成或失败，停止轮询
      if (response.status === 'COMPLETED' || response.status === 'FAILED') {
        this.stop(taskId)
        console.log(`[Poller] 任务${taskId.slice(0, 8)}已完成，停止轮询`)
        return
      }

      // 指数退避（3s → 6s → 10s）
      const currentInterval = this.intervals.get(taskId) || 3000
      const nextInterval = Math.min(currentInterval * 2, 10000)
      this.intervals.set(taskId, nextInterval)

      console.log(`[Poller] 任务${taskId.slice(0, 8)}下次轮询间隔: ${nextInterval}ms`)

      // 继续轮询
      const timerId = window.setTimeout(() => {
        this.poll(taskId, callback)
      }, nextInterval)

      this.timers.set(taskId, timerId)
    } catch (error) {
      console.error(`[Poller] 轮询任务${taskId}失败:`, error)
      // 轮询失败，停止轮询
      this.stop(taskId)
    }
  }

  /**
   * 停止轮询单个任务
   */
  stop(taskId: string) {
    const timerId = this.timers.get(taskId)
    if (timerId) {
      clearTimeout(timerId)
      this.timers.delete(taskId)
      this.intervals.delete(taskId)
    }
  }

  /**
   * 停止所有轮询
   */
  stopAll() {
    for (const timerId of this.timers.values()) {
      clearTimeout(timerId)
    }
    this.timers.clear()
    this.intervals.clear()
    console.log('[Poller] 所有轮询已停止')
  }

  /**
   * 获取当前轮询的任务数量
   */
  get activeCount(): number {
    return this.timers.size
  }
}

// 导出单例
export const taskPoller = new TaskPoller()

