// Package utils 工具函数
package utils

import "strings"

// SanitizeFilename 清理文件名中的非法字符
func SanitizeFilename(name string) string {
	// 替换 Windows 文件名中的非法字符
	illegal := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, ch := range illegal {
		result = strings.ReplaceAll(result, ch, "_")
	}
	return result
}