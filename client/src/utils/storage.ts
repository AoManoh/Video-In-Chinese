/**
 * localStorage封装工具
 * 
 * 用于持久化配置状态和任务列表
 */

import type { Task } from '@/api/types'

// Storage keys
const STORAGE_KEYS = {
  CONFIG_STATUS: 'is_configured',
  TASK_LIST: 'task_list'
} as const

// 最多保存的任务数量
const MAX_TASKS = 50

/**
 * 配置状态管理
 */
export const setConfigStatus = (isConfigured: boolean): void => {
  localStorage.setItem(STORAGE_KEYS.CONFIG_STATUS, String(isConfigured))
}

export const getConfigStatus = (): boolean => {
  return localStorage.getItem(STORAGE_KEYS.CONFIG_STATUS) === 'true'
}

/**
 * 任务列表管理
 */
export const getTaskList = (): Task[] => {
  try {
    const data = localStorage.getItem(STORAGE_KEYS.TASK_LIST)
    if (!data) return []

    const tasks: Task[] = JSON.parse(data)
    // 按创建时间倒序排列
    return tasks.sort((a, b) => b.created_at - a.created_at)
  } catch (error) {
    console.error('加载任务列表失败:', error)
    return []
  }
}

export const setTaskList = (tasks: Task[]): void => {
  try {
    // 只保存最近的50个任务
    const tasksToSave = tasks.slice(0, MAX_TASKS)
    localStorage.setItem(STORAGE_KEYS.TASK_LIST, JSON.stringify(tasksToSave))
  } catch (error) {
    console.error('保存任务列表失败:', error)
  }
}

export const addTask = (task: Task): void => {
  const tasks = getTaskList()

  // 检查是否已存在
  if (tasks.some(t => t.task_id === task.task_id)) {
    return
  }

  // 添加到列表头部
  tasks.unshift(task)
  setTaskList(tasks)
}

export const updateTask = (taskId: string, updates: Partial<Task>): void => {
  const tasks = getTaskList()
  const index = tasks.findIndex(t => t.task_id === taskId)

  if (index !== -1) {
    tasks[index] = { ...tasks[index], ...updates, updated_at: Date.now() }
    setTaskList(tasks)
  }
}

export const removeTask = (taskId: string): void => {
  const tasks = getTaskList()
  const filteredTasks = tasks.filter(t => t.task_id !== taskId)
  setTaskList(filteredTasks)
}

/**
 * 清理过期任务（7天前）
 */
export const cleanupExpiredTasks = (): void => {
  const tasks = getTaskList()
  const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000

  const activeTasks = tasks.filter(task => {
    // 保留7天内的任务，或者仍在处理中的任务
    return (
      task.created_at > sevenDaysAgo ||
      task.status === 'PROCESSING' ||
      task.status === 'PENDING'
    )
  })

  setTaskList(activeTasks)
}

