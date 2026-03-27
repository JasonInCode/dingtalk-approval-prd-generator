// Package utils 工具函数
package utils

import "time"

// TimestampToDateTime 毫秒时间戳转为日期时间字符串
func TimestampToDateTime(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.Unix(ms/1000, 0).Format("2006-01-02 15:04:05")
}