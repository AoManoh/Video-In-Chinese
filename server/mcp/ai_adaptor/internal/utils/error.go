package utils

import "strings"

// IsNonRetryableError 判断是否为不可重试的错误
// 不可重试的错误包括：
//   - 401/403: API 密钥无效
//   - 429: API 配额不足
//   - 404: 资源不存在
//
// 参数:
//   - err: 错误对象
//
// 返回:
//   - bool: true 表示不可重试，false 表示可以重试
func IsNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// 401/403: API 密钥无效
	if strings.Contains(errMsg, "API 密钥无效") ||
		strings.Contains(errMsg, "HTTP 401") ||
		strings.Contains(errMsg, "HTTP 403") {
		return true
	}

	// 429: API 配额不足
	if strings.Contains(errMsg, "API 配额不足") ||
		strings.Contains(errMsg, "HTTP 429") {
		return true
	}

	// 404: 资源不存在（需要特殊处理，但不重试当前请求）
	if strings.Contains(errMsg, "HTTP 404") {
		return true
	}

	return false
}

