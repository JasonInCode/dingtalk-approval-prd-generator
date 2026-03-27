// Package output 文件输出模块
package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"dingtalk-approval-prd-generator/model"
)

// WriteCSV 将审批表单列表写入 CSV 文件
func WriteCSV(forms []model.ApprovalForm, filename string) error {
	// 📝 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 📄 创建 CSV 写入器（使用 UTF-8 BOM 以支持 Excel 正确识别中文）
	file.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// ✏️ 写入表头
	header := []string{"类型", "流程标题", "流程编码", "描述", "状态", "修改者昵称", "修改时间", "可见范围描述"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	// 📝 写入数据行
	for _, form := range forms {
		row := []string{
			form.Category,
			form.Title,
			form.Code,
			form.Description,
			form.Status,
			form.ModifierNick,
			form.ModifiedTime,
			form.VisibleRange,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入数据失败: %w", err)
		}
	}

	return nil
}