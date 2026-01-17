package main

import (
	"fmt"
)

// AppError 应用程序错误类型
type AppError struct {
	Code    string // 错误代码
	Message string // 用户友好的错误消息
	Err     error  // 底层错误
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回底层错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// 预定义错误代码
const (
	ErrCodeInvalidArgument   = "INVALID_ARGUMENT"
	ErrCodeFileNotFound      = "FILE_NOT_FOUND"
	ErrCodeDownloadFailed    = "DOWNLOAD_FAILED"
	ErrCodeParseFailed       = "PARSE_FAILED"
	ErrCodeUnsupportedType   = "UNSUPPORTED_TYPE"
	ErrCodeNetworkError      = "NETWORK_ERROR"
)

// NewInvalidArgumentError 创建参数错误
func NewInvalidArgumentError(message string) *AppError {
	return &AppError{
		Code:    ErrCodeInvalidArgument,
		Message: message,
	}
}

// NewFileNotFoundError 创建文件未找到错误
func NewFileNotFoundError(path string, err error) *AppError {
	return &AppError{
		Code:    ErrCodeFileNotFound,
		Message: fmt.Sprintf("文件未找到: %s。请检查路径是否正确。", path),
		Err:     err,
	}
}

// NewDownloadFailedError 创建下载失败错误
func NewDownloadFailedError(uri string, err error) *AppError {
	return &AppError{
		Code:    ErrCodeDownloadFailed,
		Message: fmt.Sprintf("无法下载文件: %s。请检查 URL 是否可访问，或网络连接是否正常。", uri),
		Err:     err,
	}
}

// NewParseFailedError 创建解析失败错误
func NewParseFailedError(path string, err error) *AppError {
	return &AppError{
		Code:    ErrCodeParseFailed,
		Message: fmt.Sprintf("无法解析 profile 文件: %s。请确保文件格式正确。", path),
		Err:     err,
	}
}

// NewUnsupportedTypeError 创建不支持类型错误
func NewUnsupportedTypeError(profileType string) *AppError {
	return &AppError{
		Code:    ErrCodeUnsupportedType,
		Message: fmt.Sprintf("不支持的 profile 类型: %s。支持的类型: cpu, heap, goroutine, allocs, mutex, block", profileType),
	}
}
