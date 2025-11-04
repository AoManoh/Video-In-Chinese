/**
 * 任务管理API封装
 * 
 * @reference notes/client/2nd/API-Types.md
 */

import httpClient from '@/utils/http-client'
import type { UploadTaskResponse, GetTaskStatusResponse } from './types'

/**
 * 上传视频文件并创建任务
 * 
 * @backend POST /v1/tasks/upload
 * @reference Gateway-design.md v5.9 第289-290行
 * @param file 视频文件
 * @param onProgress 上传进度回调
 * @param signal AbortSignal 用于取消上传
 * @returns UploadTaskResponse
 */
export const uploadTask = async (
  file: File,
  onProgress?: (percent: number) => void,
  signal?: AbortSignal
): Promise<UploadTaskResponse> => {
  const formData = new FormData()
  formData.append('file', file)

  const response = await httpClient.post<UploadTaskResponse>('/v1/tasks/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    onUploadProgress: progressEvent => {
      if (onProgress && progressEvent.total) {
        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        onProgress(percent)
      }
    },
    signal
  })

  return response.data
}

/**
 * 查询任务状态
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第293-294行
 * @param taskId 任务ID
 * @returns GetTaskStatusResponse
 */
export const getTaskStatus = async (taskId: string): Promise<GetTaskStatusResponse> => {
  const response = await httpClient.get<GetTaskStatusResponse>(`/v1/tasks/${taskId}/status`)
  return response.data
}

/**
 * 下载任务结果文件
 * 
 * @backend GET /v1/tasks/download/:taskId/:fileName
 * @reference Gateway-design.md v5.9 第297-298行
 * @param taskId 任务ID
 * @param fileName 文件名
 * @returns Blob（文件内容）
 */
export const downloadFile = async (taskId: string, fileName: string): Promise<Blob> => {
  const response = await httpClient.get(`/v1/tasks/download/${taskId}/${fileName}`, {
    responseType: 'blob'
  })
  return response.data
}

/**
 * 辅助函数：触发浏览器下载
 * 
 * @param blob 文件内容
 * @param fileName 保存的文件名
 */
export const triggerDownload = (blob: Blob, fileName: string): void => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

