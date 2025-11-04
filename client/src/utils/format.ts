/**
 * 格式化工具函数
 */

/**
 * 格式化文件大小
 * 
 * @param bytes 字节数
 * @returns 格式化后的字符串（B、KB、MB、GB）
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes < 0) return '0 B'
  if (bytes === 0) return '0 B'
  if (bytes < 1024) {
    return `${bytes} B`
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`
  } else if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  } else {
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
  }
}

/**
 * 格式化时间
 * 
 * @param timestamp Unix时间戳（毫秒）
 * @returns 格式化后的字符串（YYYY-MM-DD HH:mm:ss）
 */
export const formatTime = (timestamp: number): string => {
  if (!timestamp || timestamp < 0) return '-'

  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')

  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}

/**
 * 格式化速度
 * 
 * @param bytesPerSecond 字节/秒
 * @returns 格式化后的字符串（B/s、KB/s、MB/s）
 */
export const formatSpeed = (bytesPerSecond: number): string => {
  if (bytesPerSecond < 0) return '0 B/s'
  if (bytesPerSecond === 0) return '0 B/s'
  if (bytesPerSecond < 1024) {
    return `${bytesPerSecond.toFixed(0)} B/s`
  } else if (bytesPerSecond < 1024 * 1024) {
    return `${(bytesPerSecond / 1024).toFixed(2)} KB/s`
  } else {
    return `${(bytesPerSecond / 1024 / 1024).toFixed(2)} MB/s`
  }
}

/**
 * 格式化时长
 * 
 * @param seconds 秒数
 * @returns 格式化后的字符串（Xs、Xm Ys、Xh Ym）
 */
export const formatDuration = (seconds: number): string => {
  if (seconds < 0) return '0s'
  if (seconds === 0) return '0s'
  if (seconds < 60) {
    return `${Math.floor(seconds)}s`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = Math.floor(seconds % 60)
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`
  } else {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
  }
}

