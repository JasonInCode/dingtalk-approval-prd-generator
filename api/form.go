// Package api HTTP API 客户端
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"dingtalk-approval-prd-generator/model"
)

const formDetailURL = "https://aflow.dingtalk.com/dingtalk/web/query/form/getFormAndProcessConfigV2.json"

// FetchFormDetail 获取表单详情
func FetchFormDetail(cookie, csrfToken, corpId, processCode string) (*model.FormDetailJSON, error) {
	// 🚀 创建 HTTP 客户端
	client := DefaultHTTPClient()

	// 📝 创建请求
	formData := url.Values{}
	formData.Set("processCode", processCode)
	formData.Set("tag", "")
	formData.Set("appType", "0")
	formData.Set("isCopyPublic", "false")
	formData.Set("processDraftType", "")

	req, err := http.NewRequest("POST", formDetailURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 🍪 设置 Cookie 和请求头
	SetCommonHeaders(req, cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Origin", "https://aflow.dingtalk.com")
	// 🔑 关键请求头
	req.Header.Set("_csrf_token_", csrfToken)
	req.Header.Set("x-client-corpid", corpId)

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
	var formResp model.FormDetailJSON
	if err := json.Unmarshal(body, &formResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return &formResp, nil
}

// ParseFormDetail 解析表单详情JSON文件
func ParseFormDetail(filePath string) (*model.FormDetailJSON, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var detail model.FormDetailJSON
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return &detail, nil
}