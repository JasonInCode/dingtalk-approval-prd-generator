// Package api HTTP API 客户端
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"dingtalk-approval-prd-generator/model"
)

const approvalListURL = "https://aflow.dingtalk.com/dingtalk/web/query/process/getMgrProcessList.json"

// FetchApprovalList 获取审批列表数据
func FetchApprovalList(cookie string) (*model.ApiResponse, error) {
	// 🚀 创建 HTTP 客户端
	client := DefaultHTTPClient()

	// 📝 创建请求
	reqBody := []byte("{}")
	req, err := http.NewRequest("POST", approvalListURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 🍪 设置 Cookie 和请求头
	SetCommonHeaders(req, cookie)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// 🔄 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// ✅ 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 📝 解析 JSON
	var apiResp model.ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return &apiResp, nil
}

// ToApprovalForms 将 API 响应转换为结构化的审批表单列表
func ToApprovalForms(r *model.ApiResponse, formatTime func(int64) string) []model.ApprovalForm {
	var forms []model.ApprovalForm

	for _, dir := range r.Data.SortedDirProcessList {
		for _, proc := range dir.SortedProcessAndFormVoList {
			form := model.ApprovalForm{
				Category:     dir.DirName,
				Title:        proc.FlowTitle,
				Code:         proc.ProcessCode,
				Description:  proc.Description,
				Status:       proc.ProcessStatus,
				ModifierNick: proc.ModifierNick,
				ModifiedTime: formatTime(proc.GmtModified),
				VisibleRange: proc.VisibleSummaryText,
			}
			forms = append(forms, form)
		}
	}

	return forms
}