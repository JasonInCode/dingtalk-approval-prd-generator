// Package output 文件输出模块
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteJSON 将数据写入 JSON 文件
func WriteJSON(data interface{}, filename string) error {
	// 📝 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 📄 写入 JSON（带格式化）
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("写入 JSON 失败: %w", err)
	}

	return nil
}