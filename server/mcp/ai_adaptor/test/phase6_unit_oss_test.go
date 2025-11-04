package test

import (
	"testing"

	"video-in-chinese/ai_adaptor/internal/utils"
)

// TestGenerateObjectKey 测试对象键生成
func TestGenerateObjectKey(t *testing.T) {
	tests := []struct {
		name          string
		localFilePath string
		prefix        string
		wantPattern   string // 期望的正则表达式模式
	}{
		{
			name:          "生成 ASR 音频对象键",
			localFilePath: "/path/to/audio.wav",
			prefix:        "asr-audio",
			wantPattern:   `^asr-audio/\d{4}/\d{2}/\d{2}/audio\.wav$`,
		},
		{
			name:          "生成音色参考音频对象键",
			localFilePath: "/path/to/reference.mp3",
			prefix:        "voice-reference",
			wantPattern:   `^voice-reference/\d{4}/\d{2}/\d{2}/reference\.mp3$`,
		},
		{
			name:          "Windows 路径",
			localFilePath: `C:\Users\test\audio.wav`,
			prefix:        "test",
			wantPattern:   `^test/\d{4}/\d{2}/\d{2}/audio\.wav$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectKey := utils.GenerateObjectKey(tt.localFilePath, tt.prefix)

			// 验证对象键不为空
			if objectKey == "" {
				t.Errorf("GenerateObjectKey() returned empty string")
			}

			// 验证对象键格式（使用简单的字符串检查）
			// 格式应该是: prefix/YYYY/MM/DD/filename
			if len(objectKey) < len(tt.prefix)+15 { // prefix + /YYYY/MM/DD/ + filename
				t.Errorf("GenerateObjectKey() = %v, 格式不正确", objectKey)
			}

			// 验证前缀
			if objectKey[:len(tt.prefix)] != tt.prefix {
				t.Errorf("GenerateObjectKey() prefix = %v, want %v", objectKey[:len(tt.prefix)], tt.prefix)
			}

			t.Logf("生成的对象键: %s", objectKey)
		})
	}
}

// TestNewOSSUploader_ParameterValidation 测试 OSSUploader 参数验证
func TestNewOSSUploader_ParameterValidation(t *testing.T) {
	tests := []struct {
		name            string
		accessKeyID     string
		accessKeySecret string
		endpoint        string
		bucketName      string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "所有参数为空",
			accessKeyID:     "",
			accessKeySecret: "",
			endpoint:        "",
			bucketName:      "",
			wantErr:         true,
			errContains:     "accessKeyID 不能为空",
		},
		{
			name:            "accessKeyID 为空",
			accessKeyID:     "",
			accessKeySecret: "secret",
			endpoint:        "oss-cn-shanghai.aliyuncs.com",
			bucketName:      "test-bucket",
			wantErr:         true,
			errContains:     "accessKeyID 不能为空",
		},
		{
			name:            "accessKeySecret 为空",
			accessKeyID:     "key-id",
			accessKeySecret: "",
			endpoint:        "oss-cn-shanghai.aliyuncs.com",
			bucketName:      "test-bucket",
			wantErr:         true,
			errContains:     "accessKeySecret 不能为空",
		},
		{
			name:            "endpoint 为空",
			accessKeyID:     "key-id",
			accessKeySecret: "secret",
			endpoint:        "",
			bucketName:      "test-bucket",
			wantErr:         true,
			errContains:     "endpoint 不能为空",
		},
		{
			name:            "bucketName 为空",
			accessKeyID:     "key-id",
			accessKeySecret: "secret",
			endpoint:        "oss-cn-shanghai.aliyuncs.com",
			bucketName:      "",
			wantErr:         true,
			errContains:     "bucketName 不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uploader, err := utils.NewOSSUploader(tt.accessKeyID, tt.accessKeySecret, tt.endpoint, tt.bucketName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOSSUploader() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("NewOSSUploader() error = %v, want error containing %v", err, tt.errContains)
				}
				t.Logf("预期错误: %v", err)
			} else {
				if err != nil {
					t.Errorf("NewOSSUploader() unexpected error = %v", err)
					return
				}
				if uploader == nil {
					t.Errorf("NewOSSUploader() returned nil uploader")
				}
			}
		})
	}
}

// TestNewOSSUploader_InvalidCredentials 测试无效凭证
func TestNewOSSUploader_InvalidCredentials(t *testing.T) {
	// 注意：这个测试会尝试创建 OSS 客户端，但不会实际连接到 OSS
	// 阿里云 SDK 在创建客户端时不会验证凭证，只有在实际调用 API 时才会验证

	uploader, err := utils.NewOSSUploader(
		"invalid-key-id",
		"invalid-secret",
		"oss-cn-shanghai.aliyuncs.com",
		"test-bucket",
	)

	// 创建客户端应该成功（SDK 不会在创建时验证凭证）
	if err != nil {
		t.Errorf("NewOSSUploader() with invalid credentials should succeed at creation, got error = %v", err)
		return
	}

	if uploader == nil {
		t.Errorf("NewOSSUploader() returned nil uploader")
	}

	t.Logf("OSS 客户端创建成功（凭证验证将在实际 API 调用时进行）")
}

// TestOSSUploader_UploadFile_FileNotExist 测试上传不存在的文件
func TestOSSUploader_UploadFile_FileNotExist(t *testing.T) {
	// 跳过此测试，因为需要真实的 OSS 凭证
	t.Skip("需要真实的 OSS 凭证，跳过此测试")

	uploader, err := utils.NewOSSUploader(
		"test-key-id",
		"test-secret",
		"oss-cn-shanghai.aliyuncs.com",
		"test-bucket",
	)
	if err != nil {
		t.Fatalf("NewOSSUploader() error = %v", err)
	}

	// 尝试上传不存在的文件
	_, err = uploader.UploadFile(nil, "/path/to/nonexistent/file.wav", "test/file.wav")
	if err == nil {
		t.Errorf("UploadFile() with nonexistent file should return error")
	}

	if !contains(err.Error(), "本地文件不存在") {
		t.Errorf("UploadFile() error = %v, want error containing '本地文件不存在'", err)
	}

	t.Logf("预期错误: %v", err)
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

// containsSubstring 辅助函数：检查字符串是否包含子串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

