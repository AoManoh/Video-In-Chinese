package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSUploader 封装了适配器层需要的阿里云 OSS 操作，提供上传、删除、查询等高层接口，
// 让业务代码无需直接依赖 OSS SDK 的细节或 bucket 配置方式。
type OSSUploader struct {
	client     *oss.Client
	bucketName string
}

// NewOSSUploader 创建 OSS 上传器，并对关键配置进行预检查。
//
// 参数:
//   - accessKeyID: 阿里云 AccessKey ID
//   - accessKeySecret: 阿里云 AccessKey Secret
//   - endpoint: OSS 端点（如 "oss-cn-shanghai.aliyuncs.com"）
//   - bucketName: OSS Bucket 名称
//
// 返回:
//   - *OSSUploader: OSS 上传器实例
//   - error: 错误信息
//
// 设计说明:
//   - 在初始化阶段验证 AccessKey、endpoint 与 bucket，确保错误尽早暴露。
//   - 上传器实例可复用，避免重复创建 oss.Client 并充分利用其连接池。
func NewOSSUploader(accessKeyID, accessKeySecret, endpoint, bucketName string) (*OSSUploader, error) {
	// 验证参数
	if accessKeyID == "" {
		return nil, fmt.Errorf("accessKeyID 不能为空")
	}
	if accessKeySecret == "" {
		return nil, fmt.Errorf("accessKeySecret 不能为空")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint 不能为空")
	}
	if bucketName == "" {
		return nil, fmt.Errorf("bucketName 不能为空")
	}

	// 创建 OSS 客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建 OSS 客户端失败: %w", err)
	}

	log.Printf("[OSSUploader] OSS 客户端创建成功: endpoint=%s, bucket=%s", endpoint, bucketName)

	return &OSSUploader{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile 上传文件到 OSS，并返回可公开访问的 HTTPS URL。
//
// 参数:
//   - ctx: 上下文
//   - localFilePath: 本地文件路径
//   - objectKey: OSS 对象键（如 "audio/2024/11/04/file.wav"）
//
// 返回:
//   - publicURL: 公开访问 URL
//   - error: 错误信息
//
// 设计说明:
//   - 上传前先检查文件是否存在，避免 OSS SDK 返回不直观的错误。
//   - 统一使用 https://bucket.endpoint/key 的格式，方便直接存储与前端展示。
//   - ctx 会传递给 SDK，调用方可控制超时或取消。
func (u *OSSUploader) UploadFile(ctx context.Context, localFilePath, objectKey string) (string, error) {
	log.Printf("[OSSUploader] 开始上传文件: local=%s, object_key=%s", localFilePath, objectKey)

	// 步骤 1: 验证本地文件存在
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("本地文件不存在: %s", localFilePath)
	}

	// 步骤 2: 获取 Bucket
	bucket, err := u.client.Bucket(u.bucketName)
	if err != nil {
		return "", fmt.Errorf("获取 Bucket 失败: %w", err)
	}

	// 步骤 3: 上传文件
	err = bucket.PutObjectFromFile(objectKey, localFilePath)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

	// 步骤 4: 生成公开访问 URL
	publicURL := fmt.Sprintf("https://%s.%s/%s", u.bucketName, u.client.Config.Endpoint, objectKey)

	log.Printf("[OSSUploader] 文件上传成功: url=%s", publicURL)
	return publicURL, nil
}

// GenerateObjectKey 生成 OSS 对象键。
//
// 参数:
//   - localFilePath: 本地文件路径
//   - prefix: 前缀（如 "audio", "reference"）
//
// 返回:
//   - objectKey: OSS 对象键（格式: prefix/YYYY/MM/DD/filename）
//
// 设计说明:
//   - 使用日期分片目录，避免 bucket 中单级目录过大。
//   - 函数不触发 IO 操作，重试时可复用相同的对象键。
func GenerateObjectKey(localFilePath, prefix string) string {
	// 获取文件名
	filename := filepath.Base(localFilePath)

	// 生成时间路径（YYYY/MM/DD）
	now := time.Now()
	datePath := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// 组合对象键
	objectKey := fmt.Sprintf("%s/%s/%s", prefix, datePath, filename)

	return objectKey
}

// DeleteFile 删除 OSS 文件，调用失败时返回错误以便上层决定是否重试。
//
// 参数:
//   - ctx: 上下文
//   - objectKey: OSS 对象键
//
// 返回:
//   - error: 错误信息
//
// 设计说明:
//   - 删除操作保持幂等；当目标对象不存在时，OSS 也会返回成功，适合在补偿逻辑中直接调用。
func (u *OSSUploader) DeleteFile(ctx context.Context, objectKey string) error {
	log.Printf("[OSSUploader] 删除文件: object_key=%s", objectKey)

	// 获取 Bucket
	bucket, err := u.client.Bucket(u.bucketName)
	if err != nil {
		return fmt.Errorf("获取 Bucket 失败: %w", err)
	}

	// 删除文件
	err = bucket.DeleteObject(objectKey)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	log.Printf("[OSSUploader] 文件删除成功: object_key=%s", objectKey)
	return nil
}

// FileExists 检查文件是否存在，用于补偿逻辑或缓存验证场景。
//
// 参数:
//   - ctx: 上下文
//   - objectKey: OSS 对象键
//
// 返回:
//   - exists: 文件是否存在
//   - error: 错误信息
//
// 设计说明:
//   - 调用 oss.IsObjectExist 仅检查对象元数据，不下载实际内容。
//   - 若发生网络或权限问题，会保留原始错误便于上游日志排查。
func (u *OSSUploader) FileExists(ctx context.Context, objectKey string) (bool, error) {
	// 获取 Bucket
	bucket, err := u.client.Bucket(u.bucketName)
	if err != nil {
		return false, fmt.Errorf("获取 Bucket 失败: %w", err)
	}

	// 检查文件是否存在
	exists, err := bucket.IsObjectExist(objectKey)
	if err != nil {
		return false, fmt.Errorf("检查文件是否存在失败: %w", err)
	}

	return exists, nil
}
