// Package output 文件输出模块
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dingtalk-approval-prd-generator/model"
)

// GetComponentLabel 获取组件中文标签
func GetComponentLabel(componentName string) string {
	if label, ok := model.ComponentNameMap[componentName]; ok {
		return label
	}
	return componentName
}

// GetPropString 获取props中的字符串值
func GetPropString(props map[string]interface{}, key string) string {
	if val, ok := props[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case bool:
			if v {
				return "是"
			}
			return "否"
		}
	}
	return ""
}

// GetPropStringArray 获取props中的字符串数组值（用于DDDateRangeField的label字段）
func GetPropStringArray(props map[string]interface{}, key string) []string {
	if val, ok := props[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// GetPropBool 获取props中的布尔值
func GetPropBool(props map[string]interface{}, key string) bool {
	if val, ok := props[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// BuildFieldContext 构建字段上下文（并发安全）
func BuildFieldContext(items []model.PRDFormItem) *model.FieldContext {
	ctx := &model.FieldContext{
		OptionsMap:   make(map[string]map[string]string),
		IdToLabelMap: make(map[string]string),
		OptionsList:  make(map[string][]string),
	}

	for idx, item := range items {
		props := item.Props
		fieldId := GetPropString(props, "id")
		label := GetPropString(props, "label")
		if fieldId == "" {
			continue
		}

		// 特殊处理：DDDateRangeField的label是数组格式 ["开始时间", "结束时间"]
		if item.ComponentName == "DDDateRangeField" && label == "" {
			if labelArr := GetPropStringArray(props, "label"); len(labelArr) > 0 {
				label = strings.Join(labelArr, "/")
			}
		}

		// 构建字段ID到名称映射
		// 对于没有label的字段（如说明文字），使用类型+序号作为名称
		if label == "" {
			componentLabel := GetComponentLabel(item.ComponentName)
			label = fmt.Sprintf("%s(%d)", componentLabel, idx+1)
		}
		ctx.IdToLabelMap[fieldId] = label

		options := make(map[string]string)
		optionList := make([]string, 0)
		if opts, ok := props["options"].([]interface{}); ok {
			for _, opt := range opts {
				if optMap, ok := opt.(map[string]interface{}); ok {
					key, _ := optMap["key"].(string)
					value, _ := optMap["value"].(string)
					if key != "" && value != "" {
						options[key] = value
						optionList = append(optionList, value)
					}
				}
			}
		}

		if len(options) > 0 {
			ctx.OptionsMap[fieldId] = options
			ctx.OptionsList[fieldId] = optionList
		}

		// 处理子字段（TableField、DDBizSuite）
		if len(item.Children) > 0 {
			childCtx := BuildFieldContext(item.Children)
			for k, v := range childCtx.OptionsMap {
				ctx.OptionsMap[k] = v
			}
			for k, v := range childCtx.IdToLabelMap {
				ctx.IdToLabelMap[k] = v
			}
			for k, v := range childCtx.OptionsList {
				ctx.OptionsList[k] = v
			}
		}
	}

	return ctx
}

// ParseFormula 解析计算公式，将字段ID翻译为字段名称
func ParseFormula(ctx *model.FieldContext, formula interface{}) string {
	switch v := formula.(type) {
	case []interface{}:
		var parts []string
		var numBuffer strings.Builder // 用于合并连续的数字和小数点

		// 辅助函数：刷新数字缓冲区
		flushNumBuffer := func() {
			if numBuffer.Len() > 0 {
				parts = append(parts, numBuffer.String())
				numBuffer.Reset()
			}
		}

		for _, item := range v {
			switch itemVal := item.(type) {
			case string:
				// 检查是否是字段ID
				if label, ok := ctx.IdToLabelMap[itemVal]; ok {
					flushNumBuffer()
					parts = append(parts, label)
				} else if itemVal == "." {
					// 小数点，追加到数字缓冲区
					numBuffer.WriteString(itemVal)
				} else {
					// 操作符（+, -, *, / 等），先刷新数字缓冲区
					flushNumBuffer()
					parts = append(parts, itemVal)
				}
			case float64:
				// JSON数字默认解析为float64，追加到数字缓冲区
				if itemVal == float64(int64(itemVal)) {
					numBuffer.WriteString(fmt.Sprintf("%d", int64(itemVal)))
				} else {
					numBuffer.WriteString(fmt.Sprintf("%g", itemVal))
				}
			case int:
				// 整数类型，追加到数字缓冲区
				numBuffer.WriteString(fmt.Sprintf("%d", itemVal))
			case map[string]interface{}:
				// 字段引用对象
				flushNumBuffer()
				if id, ok := itemVal["id"].(string); ok {
					if label, exists := ctx.IdToLabelMap[id]; exists {
						parts = append(parts, label)
					} else {
						parts = append(parts, id)
					}
				}
			}
		}
		// 刷新剩余的数字缓冲区
		flushNumBuffer()
		return strings.Join(parts, " ")
	case string:
		return v
	}
	return ""
}

// TranslateOptionValue 将选项key翻译为中文
func TranslateOptionValue(ctx *model.FieldContext, paramKey, optionKey string) string {
	// 尝试直接匹配
	if options, ok := ctx.OptionsMap[paramKey]; ok {
		if label, exists := options[optionKey]; exists {
			return label
		}
	}

	// 如果直接匹配失败，尝试按索引匹配（处理钉钉数据不一致的情况）
	// 例如 option_0 对应第一个选项，option_1 对应第二个选项
	if strings.HasPrefix(optionKey, "option_") {
		// 提取索引
		indexStr := strings.TrimPrefix(optionKey, "option_")
		// 尝试解析索引（可能是数字或其他格式）
		if optionList, ok := ctx.OptionsList[paramKey]; ok && len(optionList) > 0 {
			// 尝试将索引转换为数字
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err == nil {
				if index >= 0 && index < len(optionList) {
					return optionList[index]
				}
			}
		}
	}

	// 无法翻译，返回原始key
	return optionKey
}

// ActionerRoleInfo 审批角色完整信息
type ActionerRoleInfo struct {
	RoleName        string // 角色名称
	ActType         string // 会签/或签: "and"=会签, "or"=或签
	IsSelfSelect    bool   // 是否发起人自选
	SelfSelectScope string // 发起人自选范围（全体员工/指定标签等）
}

// extractActionerRoleInfo 从单个actionerRule中提取完整审批角色信息
func extractActionerRoleInfo(rule map[string]interface{}) *ActionerRoleInfo {
	info := &ActionerRoleInfo{}

	// 提取会签/或签类型
	if actType, ok := rule["actType"].(string); ok {
		info.ActType = actType
	}

	// 检查审批人类型
	actionType, hasType := rule["type"].(string)

	// 处理发起人自选类型
	if hasType && actionType == "target_select" {
		info.IsSelfSelect = true
		// 检查选择范围
		if selectArr, ok := rule["select"].([]interface{}); ok {
			for _, s := range selectArr {
				if s == "allStaff" {
					info.SelfSelectScope = "全体员工"
				} else if s == "labels" {
					// 从range.labels中获取标签名称
					if range_ := rule["range"]; range_ != nil {
						if rangeMap, ok := range_.(map[string]interface{}); ok {
							if labels := rangeMap["labels"]; labels != nil {
								if labelList, ok := labels.([]interface{}); ok && len(labelList) > 0 {
									names := make([]string, 0)
									for _, l := range labelList {
										if labelMap, ok := l.(map[string]interface{}); ok {
											if name, ok := labelMap["labelNames"].(string); ok && name != "" {
												names = append(names, name)
											}
										}
									}
									if len(names) > 0 {
										info.SelfSelectScope = strings.Join(names, "、")
									}
								}
							}
						}
					}
				} else if s == "approvals" {
					// 从指定人员列表中自选
					if range_ := rule["range"]; range_ != nil {
						if rangeMap, ok := range_.(map[string]interface{}); ok {
							if approvals := rangeMap["approvals"]; approvals != nil {
								if approvalList, ok := approvals.([]interface{}); ok && len(approvalList) > 0 {
									names := make([]string, 0)
									for _, a := range approvalList {
										if approvalMap, ok := a.(map[string]interface{}); ok {
											if name, ok := approvalMap["userName"].(string); ok && name != "" {
												names = append(names, name)
											}
										}
									}
									if len(names) > 0 {
										info.SelfSelectScope = strings.Join(names, "、")
									}
								}
							}
						}
					}
				}
			}
		}
		return info
	}

	// 优先从labelNames获取（最常见：target_label类型）
	if labelNames, ok := rule["labelNames"].(string); ok && labelNames != "" {
		info.RoleName = labelNames
		return info
	}

	// 从label获取（表单组件审批人）
	if label, ok := rule["label"].(string); ok && label != "" {
		info.RoleName = label
		return info
	}

	// 处理其他类型
	if hasType {
		switch actionType {
		case "target_management":
			level := 1
			if l, ok := rule["level"].(float64); ok {
				level = int(l)
			}
			if level == 1 {
				info.RoleName = "直属上级"
			} else {
				info.RoleName = fmt.Sprintf("第%d级上级", level)
			}
			return info
		case "target_originator":
			info.RoleName = "发起人"
			return info
		case "self_select":
			info.IsSelfSelect = true
			return info
		case "leader":
			info.RoleName = "部门主管"
			return info
		case "target_approval":
			// 指定具体审批人，从approvals中提取人员名称
			if approvals, ok := rule["approvals"].([]interface{}); ok && len(approvals) > 0 {
				names := make([]string, 0)
				for _, a := range approvals {
					if approvalMap, ok := a.(map[string]interface{}); ok {
						if name, ok := approvalMap["userName"].(string); ok && name != "" {
							names = append(names, name)
						}
					}
				}
				if len(names) > 0 {
					info.RoleName = strings.Join(names, "、")
				}
			}
			return info
		}
	}

	// 从range.labels中获取
	if range_ := rule["range"]; range_ != nil {
		if rangeMap, ok := range_.(map[string]interface{}); ok {
			if labels := rangeMap["labels"]; labels != nil {
				if labelList, ok := labels.([]interface{}); ok && len(labelList) > 0 {
					names := make([]string, 0)
					for _, l := range labelList {
						if labelMap, ok := l.(map[string]interface{}); ok {
							if name, ok := labelMap["labelNames"].(string); ok && name != "" {
								names = append(names, name)
							}
						}
					}
					if len(names) > 0 {
						info.RoleName = strings.Join(names, "、")
						return info
					}
				}
			}
		}
	}

	return info
}

// formatActionerRoleInfo 格式化审批角色信息输出
func formatActionerRoleInfo(info *ActionerRoleInfo, rolePrefix string) string {
	if info == nil {
		return ""
	}

	var parts []string

	// 处理发起人自选
	if info.IsSelfSelect {
		if info.SelfSelectScope != "" {
			parts = append(parts, fmt.Sprintf("发起人自选（可选范围：%s）", info.SelfSelectScope))
		} else {
			parts = append(parts, "发起人自选")
		}
	} else if info.RoleName != "" {
		parts = append(parts, info.RoleName)
	}

	// 添加会签/或签信息
	if info.ActType == "and" {
		parts = append(parts, "会签")
	} else if info.ActType == "or" {
		parts = append(parts, "或签")
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s: %s", rolePrefix, strings.Join(parts, "，"))
}

// isGenericNodeName 判断是否为通用占位符名称（需要从actionerRules中提取详细信息）
func isGenericNodeName(name string) bool {
	genericNames := []string{"UNKNOWN", "审批人", "抄送人", "审批", "抄送", "审核人", "审核"}
	for _, generic := range genericNames {
		if name == generic {
			return true
		}
	}
	return false
}

// hasApprovalNodes 检查流程是否有实际的审批/抄送/审核节点
func hasApprovalNodes(node *model.PRDProcessNode) bool {
	if node == nil {
		return false
	}

	// 检查当前节点类型
	switch node.Type {
	case "approver", "notifier", "audit", "payment":
		return true
	case "route":
		// 检查分支条件节点
		for i := range node.ConditionNodes {
			if hasApprovalNodes(&node.ConditionNodes[i]) {
				return true
			}
		}
	}

	// 递归检查子节点
	if node.ChildNode != nil {
		return hasApprovalNodes(node.ChildNode)
	}

	return false
}

// extractActionerInfo 从actionerRules中提取审批人/抄送人详细信息
func extractActionerInfo(rules []interface{}) string {
	names := make([]string, 0)
	for _, r := range rules {
		if rule, ok := r.(map[string]interface{}); ok {
			// 检查审批人类型
			actionType, hasType := rule["type"].(string)

			// 优先使用labelNames
			if labelNames, ok := rule["labelNames"].(string); ok && labelNames != "" {
				names = append(names, labelNames)
			} else if label, ok := rule["label"].(string); ok && label != "" {
				// 处理target_formcomponent_approval类型（表单组件审批人）
				names = append(names, label)
			} else if hasType && actionType == "target_management" {
				// 处理target_management类型（上级主管）
				level := 1
				if l, ok := rule["level"].(float64); ok {
					level = int(l)
				}
				if level == 1 {
					names = append(names, "直属上级")
				} else {
					names = append(names, fmt.Sprintf("第%d级上级", level))
				}
			} else if hasType && actionType == "target_originator" {
				// 处理target_originator类型（发起人自己）
				names = append(names, "发起人")
			}
			// 处理select字段（全体员工等）- 需要区分自动抄送全体员工 vs 发起人自选范围
			if selectArr, ok := rule["select"].([]interface{}); ok {
				for _, s := range selectArr {
					if s == "allStaff" {
						// 检查range.allStaff是否有实际值
						if range_ := rule["range"]; range_ != nil {
							if rangeMap, ok := range_.(map[string]interface{}); ok {
								if allStaff := rangeMap["allStaff"]; allStaff != nil {
									// range.allStaff有值，表示自动抄送全体员工
									names = append(names, "全体员工")
								} else {
									// range.allStaff为null，表示发起人可从全体员工中选择
									names = append(names, "发起人自选（可选范围：全体员工）")
								}
							}
						} else {
							// 无range，默认当作发起人自选
							names = append(names, "发起人自选（可选范围：全体员工）")
						}
					}
				}
			}
			// 检查range.labels[].labelNames
			if range_ := rule["range"]; range_ != nil {
				if rangeMap, ok := range_.(map[string]interface{}); ok {
					if labels := rangeMap["labels"]; labels != nil {
						if labelList, ok := labels.([]interface{}); ok && len(labelList) > 0 {
							for _, l := range labelList {
								if labelMap, ok := l.(map[string]interface{}); ok {
									if name, ok := labelMap["labelNames"].(string); ok && name != "" {
										names = append(names, name)
									}
								}
							}
						}
					}
				}
			}
			// 处理target_approval类型（指定具体审批人）
			if approvals, ok := rule["approvals"].([]interface{}); ok && len(approvals) > 0 {
				for _, a := range approvals {
					if approval, ok := a.(map[string]interface{}); ok {
						if userName, ok := approval["userName"].(string); ok && userName != "" {
							names = append(names, userName)
						}
					}
				}
			}
		}
	}
	if len(names) > 0 {
		return strings.Join(names, "、")
	}
	return ""
}

// generateFlowChart 生成流程图文本
func generateFlowChart(ctx *model.FieldContext, node *model.PRDProcessNode, depth int) string {
	var sb strings.Builder
	indent := strings.Repeat("  ", depth)

	if node == nil {
		return ""
	}

	switch node.Type {
	case "start":
		sb.WriteString(fmt.Sprintf("%s🚀 【开始】%s\n", indent, node.Name))
	case "approver":
		nodeName := node.Name
		// 如果节点名称是通用占位符或为空，尝试从actionerRules获取详细信息
		if isGenericNodeName(nodeName) || nodeName == "" {
			if node.Properties != nil {
				if rules, ok := node.Properties["actionerRules"].([]interface{}); ok && len(rules) > 0 {
					if extracted := extractActionerInfo(rules); extracted != "" {
						nodeName = extracted
					}
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s👤 【审批】%s\n", indent, nodeName))
		// 输出审批人规则
		if node.Properties != nil {
			if rules, ok := node.Properties["actionerRules"].([]interface{}); ok {
				for _, r := range rules {
					if rule, ok := r.(map[string]interface{}); ok {
						info := extractActionerRoleInfo(rule)
						if formatted := formatActionerRoleInfo(info, "审批角色"); formatted != "" {
							sb.WriteString(fmt.Sprintf("%s    └─ %s\n", indent, formatted))
						}
					}
				}
			}
		}
	case "notifier":
		// 抄送人节点
		nodeName := node.Name
		// 如果节点名称是通用占位符或为空，尝试从actionerRules获取详细信息
		if isGenericNodeName(nodeName) || nodeName == "" {
			if node.Properties != nil {
				if rules, ok := node.Properties["actionerRules"].([]interface{}); ok && len(rules) > 0 {
					if extracted := extractActionerInfo(rules); extracted != "" {
						nodeName = extracted
					}
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s📧 【抄送】%s\n", indent, nodeName))
	case "route":
		sb.WriteString(fmt.Sprintf("%s🔀 【分支】\n", indent))
		for i, cond := range node.ConditionNodes {
			condName := cond.Name
			if condName == "" {
				condName = fmt.Sprintf("分支%d", i+1)
			}
			// 解析条件
			condDesc := parseCondition(ctx, cond.Properties)
			if condDesc != "" {
				sb.WriteString(fmt.Sprintf("%s  📌 %s: %s\n", indent, condName, condDesc))
			} else {
				sb.WriteString(fmt.Sprintf("%s  📌 %s (默认)\n", indent, condName))
			}
			if cond.ChildNode != nil {
				sb.WriteString(generateFlowChart(ctx, cond.ChildNode, depth+2))
			}
		}
		// 处理route节点的childNode（默认分支）
		if node.ChildNode != nil {
			sb.WriteString(fmt.Sprintf("%s  📌 默认分支\n", indent))
			sb.WriteString(generateFlowChart(ctx, node.ChildNode, depth+2))
		}
	case "audit":
		// 审核节点（类似审批但通常用于确认/核实）
		nodeName := node.Name
		// 如果节点名称是通用占位符或为空，尝试从actionerRules获取详细信息
		if isGenericNodeName(nodeName) || nodeName == "" {
			if node.Properties != nil {
				if rules, ok := node.Properties["actionerRules"].([]interface{}); ok && len(rules) > 0 {
					if extracted := extractActionerInfo(rules); extracted != "" {
						nodeName = extracted
					}
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s✅ 【审核】%s\n", indent, nodeName))
		// 输出审核人规则
		if node.Properties != nil {
			if rules, ok := node.Properties["actionerRules"].([]interface{}); ok {
				for _, r := range rules {
					if rule, ok := r.(map[string]interface{}); ok {
						info := extractActionerRoleInfo(rule)
						if formatted := formatActionerRoleInfo(info, "审核角色"); formatted != "" {
							sb.WriteString(fmt.Sprintf("%s    └─ %s\n", indent, formatted))
						}
					}
				}
			}
		}
	case "payment":
		// 付款确认节点（智能财务专用）
		nodeName := node.Name
		if nodeName == "UNKNOWN" || nodeName == "" {
			nodeName = "付款确认"
		}
		sb.WriteString(fmt.Sprintf("%s💰 【付款】%s\n", indent, nodeName))
	case "condition":
		// 条件节点在route中已处理
	}

	// 递归处理子节点（route节点的childNode已在route case中处理，跳过）
	if node.ChildNode != nil && node.Type != "route" {
		sb.WriteString(generateFlowChart(ctx, node.ChildNode, depth))
	}

	return sb.String()
}

// parseCondition 解析条件表达式
func parseCondition(ctx *model.FieldContext, props map[string]interface{}) string {
	if props == nil {
		return ""
	}

	conditions, ok := props["conditions"].([]interface{})
	if !ok || len(conditions) == 0 {
		return ""
	}

	var results []string
	for _, cond := range conditions {
		// 条件可能是数组嵌套
		if condArr, ok := cond.([]interface{}); ok {
			for _, c := range condArr {
				if condMap, ok := c.(map[string]interface{}); ok {
					desc := buildConditionDesc(ctx, condMap)
					if desc != "" {
						results = append(results, desc)
					}
				}
			}
		} else if condMap, ok := cond.(map[string]interface{}); ok {
			desc := buildConditionDesc(ctx, condMap)
			if desc != "" {
				results = append(results, desc)
			}
		}
	}

	if len(results) > 0 {
		return strings.Join(results, " 且 ")
	}
	return ""
}

// buildConditionDesc 构建条件描述
func buildConditionDesc(ctx *model.FieldContext, cond map[string]interface{}) string {
	paramKey, _ := cond["paramKey"].(string)
	paramLabel, _ := cond["paramLabel"].(string)
	condType, _ := cond["type"].(string)

	if paramLabel == "" {
		return ""
	}

	switch condType {
	case "dingtalk_actioner_value_condition":
		// 值条件 - 翻译选项值
		if values, ok := cond["paramValues"].([]interface{}); ok && len(values) > 0 {
			vals := make([]string, 0)
			for _, v := range values {
				if vs, ok := v.(string); ok {
					// 翻译选项值
					translated := TranslateOptionValue(ctx, paramKey, vs)
					vals = append(vals, translated)
				}
			}
			return fmt.Sprintf("%s = [%s]", paramLabel, strings.Join(vals, ", "))
		}
	case "dingtalk_actioner_range_condition":
		// 范围条件 - 数值/金额条件
		lowerBound := cond["lowerBound"]
		upperBound := cond["upperBound"]
		unit, _ := cond["unit"].(string)

		// 格式化数值（避免科学计数法，支持字段引用）
		formatAmount := func(val interface{}) string {
			switch v := val.(type) {
			case float64:
				if v >= 10000 {
					return fmt.Sprintf("%.0f", v)
				}
				return fmt.Sprintf("%.0f", v)
			case int:
				return fmt.Sprintf("%d", v)
			case string:
				// 可能是字段引用，尝试翻译
				if label, ok := ctx.IdToLabelMap[v]; ok {
					return label
				}
				return v
			case map[string]interface{}:
				// 字段引用对象
				if id, ok := v["id"].(string); ok {
					if label, exists := ctx.IdToLabelMap[id]; exists {
						return label
					}
					return id
				}
				return fmt.Sprintf("%v", val)
			default:
				return fmt.Sprintf("%v", val)
			}
		}

		// 如果没有单位，尝试从参数标签推断
		if unit == "" {
			if strings.Contains(paramLabel, "天") {
				unit = "天"
			} else if strings.Contains(paramLabel, "金额") || strings.Contains(paramLabel, "元") {
				unit = "元"
			} else {
				unit = "元" // 默认单位
			}
		}

		if lowerBound != nil && upperBound != nil {
			return fmt.Sprintf("%s ≥ %s %s", paramLabel, formatAmount(lowerBound), unit)
		} else if lowerBound != nil {
			return fmt.Sprintf("%s ≥ %s %s", paramLabel, formatAmount(lowerBound), unit)
		} else if upperBound != nil {
			return fmt.Sprintf("%s ≤ %s %s", paramLabel, formatAmount(upperBound), unit)
		}
	case "dingtalk_actioner_dept_condition":
		// 部门条件
		if conds, ok := cond["conds"].([]interface{}); ok && len(conds) > 0 {
			for _, c := range conds {
				if cMap, ok := c.(map[string]interface{}); ok {
					if attrs, ok := cMap["attrs"].(map[string]interface{}); ok {
						if name, ok := attrs["name"].(string); ok {
							return fmt.Sprintf("%s 为「%s」", paramLabel, name)
						}
					}
				}
			}
			return fmt.Sprintf("%s 属于指定部门", paramLabel)
		}
	}

	return paramLabel
}

// GeneratePRD 生成PRD文档
func GeneratePRD(detail *model.FormDetailJSON, extraInfo *model.PRDExtraInfo, outputPath string) error {
	var sb strings.Builder

	// 解析表单内容
	var formContent model.FormContent
	if err := json.Unmarshal([]byte(detail.Data.FormVo.Content), &formContent); err != nil {
		return fmt.Errorf("解析表单内容失败: %w", err)
	}

	// 构建字段上下文（并发安全，每个请求独立）
	ctx := BuildFieldContext(formContent.Items)

	// 解析流程配置
	var processNode model.PRDProcessNode
	if err := json.Unmarshal([]byte(detail.Data.ProcessConfig), &processNode); err != nil {
		return fmt.Errorf("解析流程配置失败: %w", err)
	}

	// 📄 文档头部
	sb.WriteString("# PRD 产品需求文档\n\n")
	sb.WriteString("---\n\n")
	sb.WriteString("## 一、基本信息\n\n")
	sb.WriteString("| 项目 | 内容 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 表单名称 | %s |\n", formContent.Title))
	sb.WriteString(fmt.Sprintf("| 表单描述 | %s |\n", formContent.Description))
	sb.WriteString(fmt.Sprintf("| 表单类型 | %s |\n", extraInfo.Category))
	sb.WriteString(fmt.Sprintf("| 流程编码 | %s |\n", detail.Data.ProcessCode))
	sb.WriteString(fmt.Sprintf("| 流程状态 | %s |\n", detail.Data.ProcessStatus))
	sb.WriteString(fmt.Sprintf("| 流程版本 | V%s |\n", detail.Data.ProcessVersion))
	sb.WriteString(fmt.Sprintf("| 最后修改人 | %s |\n", detail.Data.ModifierName))
	sb.WriteString(fmt.Sprintf("| 修改时间 | %s |\n", extraInfo.ModifiedTime))
	sb.WriteString(fmt.Sprintf("| 可见范围 | %s |\n", extraInfo.VisibleRange))
	sb.WriteString("\n---\n\n")

	// 📝 表单字段
	sb.WriteString("## 二、表单字段设计\n\n")
	sb.WriteString("### 2.1 字段列表\n\n")
	sb.WriteString("| 序号 | 字段类型 | 字段名称 | 是否必填 | 字段说明 | 字段联动 |\n")
	sb.WriteString("|------|----------|----------|----------|----------|----------|\n")

	// 构建字段ID到字段名称的映射（用于联动显示）
	fieldIdToLabel := make(map[string]string)
	for _, item := range formContent.Items {
		fieldId := GetPropString(item.Props, "id")
		label := GetPropString(item.Props, "label")
		if fieldId != "" && label != "" {
			fieldIdToLabel[fieldId] = label
		}
	}

	for i, item := range formContent.Items {
		props := item.Props
		label := GetPropString(props, "label")
		required := GetPropBool(props, "required")
		componentLabel := GetComponentLabel(item.ComponentName)
		fieldId := GetPropString(props, "id")

		// 特殊处理：DDDateRangeField的label是数组格式 ["开始时间", "结束时间"]
		if item.ComponentName == "DDDateRangeField" && label == "" {
			if labelArr := GetPropStringArray(props, "label"); len(labelArr) > 0 {
				label = strings.Join(labelArr, "/")
			}
		}

		// 构建字段说明
		var desc strings.Builder
		if placeholder := GetPropString(props, "placeholder"); placeholder != "" {
			desc.WriteString(fmt.Sprintf("提示: %s", placeholder))
		}

		// 下拉选项（完整显示，不截取）
		if options, ok := props["options"].([]interface{}); ok && len(options) > 0 {
			if desc.Len() > 0 {
				desc.WriteString("; ")
			}
			desc.WriteString("选项: ")
			opts := make([]string, 0)
			for _, opt := range options {
				if optMap, ok := opt.(map[string]interface{}); ok {
					if val, ok := optMap["value"].(string); ok {
						opts = append(opts, val)
					}
				}
			}
			// 完整显示所有选项，不截取
			desc.WriteString(strings.Join(opts, ", "))
		}

		// 说明文字内容
		if item.ComponentName == "TextNote" {
			if content := GetPropString(props, "content"); content != "" {
				desc.WriteString(fmt.Sprintf("内容: %s", content))
			}
		}

		// 关联表单
		if item.ComponentName == "RelateField" {
			if templates, ok := props["availableTemplates"].([]interface{}); ok && len(templates) > 0 {
				if desc.Len() > 0 {
					desc.WriteString("; ")
				}
				desc.WriteString("可关联: ")
				tpls := make([]string, 0)
				for _, t := range templates {
					if tMap, ok := t.(map[string]interface{}); ok {
						if name, ok := tMap["name"].(string); ok {
							tpls = append(tpls, name)
						}
					}
				}
				desc.WriteString(strings.Join(tpls, ", "))
			}
		}

		// 计算公式
		if item.ComponentName == "CalculateField" {
			if formula, ok := props["formula"]; ok && formula != nil {
				if desc.Len() > 0 {
					desc.WriteString("; ")
				}
				formulaStr := ParseFormula(ctx, formula)
				desc.WriteString(fmt.Sprintf("公式: %s", formulaStr))
			}
		}

		// DDBizSuite 业务套件 - 展开子字段
		if item.ComponentName == "DDBizSuite" {
			if len(item.Children) > 0 {
				if desc.Len() > 0 {
					desc.WriteString("; ")
				}
				desc.WriteString("包含字段: ")
				childLabels := make([]string, 0)
				for _, child := range item.Children {
					if childLabel := GetPropString(child.Props, "label"); childLabel != "" {
						childLabels = append(childLabels, childLabel)
					}
				}
				desc.WriteString(strings.Join(childLabels, "、"))
			}
		}

		// TableField 明细表格 - 展开子字段
		if item.ComponentName == "TableField" {
			if len(item.Children) > 0 {
				if desc.Len() > 0 {
					desc.WriteString("; ")
				}
				desc.WriteString("包含字段: ")
				childLabels := make([]string, 0)
				for _, child := range item.Children {
					if childLabel := GetPropString(child.Props, "label"); childLabel != "" {
						childLabels = append(childLabels, childLabel)
					}
				}
				desc.WriteString(strings.Join(childLabels, "、"))
			}
		}

		// 构建字段联动说明
		var linkage strings.Builder
		if behaviorLinkage, ok := props["behaviorLinkage"].([]interface{}); ok && len(behaviorLinkage) > 0 {
			for _, bl := range behaviorLinkage {
				if blMap, ok := bl.(map[string]interface{}); ok {
					if value, ok := blMap["value"].(string); ok {
						// 翻译选项值
						translatedValue := TranslateOptionValue(ctx, fieldId, value)
						// 解析目标字段
						targetFields := make([]string, 0)
						if targets, ok := blMap["targets"].([]interface{}); ok {
							for _, t := range targets {
								if tMap, ok := t.(map[string]interface{}); ok {
									if targetFieldId, ok := tMap["fieldId"].(string); ok {
										if targetLabel, exists := ctx.IdToLabelMap[targetFieldId]; exists {
											targetFields = append(targetFields, targetLabel)
										} else {
											targetFields = append(targetFields, targetFieldId)
										}
									}
								}
							}
						}
						if len(targetFields) > 0 {
							linkage.WriteString(fmt.Sprintf("选「%s」时显示「%s」<br>", translatedValue, strings.Join(targetFields, "、")))
						} else {
							linkage.WriteString(fmt.Sprintf("选「%s」时显隐字段<br>", translatedValue))
						}
					}
				}
			}
		}

		requiredStr := "否"
		if required {
			requiredStr = "✅ 是"
		}

		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
			i+1, componentLabel, label, requiredStr, desc.String(), strings.TrimSuffix(linkage.String(), "<br>")))

		// 展开DDBizSuite子字段（缩进显示）
		if item.ComponentName == "DDBizSuite" && len(item.Children) > 0 {
			for j, child := range item.Children {
				childLabel := GetPropString(child.Props, "label")
				childRequired := GetPropBool(child.Props, "required")
				childComponentLabel := GetComponentLabel(child.ComponentName)

				// 构建子字段说明
				var childDesc strings.Builder
				if childPlaceholder := GetPropString(child.Props, "placeholder"); childPlaceholder != "" {
					childDesc.WriteString(fmt.Sprintf("提示: %s", childPlaceholder))
				}
				// 子字段的选项
				if childOptions, ok := child.Props["options"].([]interface{}); ok && len(childOptions) > 0 {
					if childDesc.Len() > 0 {
						childDesc.WriteString("; ")
					}
					childDesc.WriteString("选项: ")
					opts := make([]string, 0)
					for _, opt := range childOptions {
						if optMap, ok := opt.(map[string]interface{}); ok {
							if val, ok := optMap["value"].(string); ok {
								opts = append(opts, val)
							}
						}
					}
					childDesc.WriteString(strings.Join(opts, ", "))
				}

				childRequiredStr := "否"
				if childRequired {
					childRequiredStr = "✅ 是"
				}

				sb.WriteString(fmt.Sprintf("| %d.%d | └─ %s | %s | %s | %s |  |\n",
					i+1, j+1, childComponentLabel, childLabel, childRequiredStr, childDesc.String()))
			}
		}

		// 展开TableField子字段（缩进显示）
		if item.ComponentName == "TableField" && len(item.Children) > 0 {
			for j, child := range item.Children {
				childLabel := GetPropString(child.Props, "label")
				childRequired := GetPropBool(child.Props, "required")
				childComponentLabel := GetComponentLabel(child.ComponentName)

				// 特殊处理：DDDateRangeField的label是数组格式 ["开始时间", "结束时间"]
				if child.ComponentName == "DDDateRangeField" && childLabel == "" {
					if labelArr := GetPropStringArray(child.Props, "label"); len(labelArr) > 0 {
						childLabel = strings.Join(labelArr, "/")
					}
				}

				// 构建子字段说明
				var childDesc strings.Builder
				if childPlaceholder := GetPropString(child.Props, "placeholder"); childPlaceholder != "" {
					childDesc.WriteString(fmt.Sprintf("提示: %s", childPlaceholder))
				}
				// 子字段的选项
				if childOptions, ok := child.Props["options"].([]interface{}); ok && len(childOptions) > 0 {
					if childDesc.Len() > 0 {
						childDesc.WriteString("; ")
					}
					childDesc.WriteString("选项: ")
					opts := make([]string, 0)
					for _, opt := range childOptions {
						if optMap, ok := opt.(map[string]interface{}); ok {
							if val, ok := optMap["value"].(string); ok {
								opts = append(opts, val)
							}
						}
					}
					childDesc.WriteString(strings.Join(opts, ", "))
				}
				// 子字段的说明文字内容
				if child.ComponentName == "TextNote" {
					if content := GetPropString(child.Props, "content"); content != "" {
						if childDesc.Len() > 0 {
							childDesc.WriteString("; ")
						}
						childDesc.WriteString(fmt.Sprintf("内容: %s", content))
					}
				}

				childRequiredStr := "否"
				if childRequired {
					childRequiredStr = "✅ 是"
				}

				sb.WriteString(fmt.Sprintf("| %d.%d | └─ %s | %s | %s | %s |  |\n",
					i+1, j+1, childComponentLabel, childLabel, childRequiredStr, childDesc.String()))
			}
		}

		// 展开DDBizSuite子字段中的TableField子字段（嵌套展开）
		if item.ComponentName == "DDBizSuite" && len(item.Children) > 0 {
			subIdx := 0
			for _, child := range item.Children {
				if child.ComponentName == "TableField" && len(child.Children) > 0 {
					for _, tableChild := range child.Children {
						subIdx++
						tableChildLabel := GetPropString(tableChild.Props, "label")
						tableChildRequired := GetPropBool(tableChild.Props, "required")
						tableChildComponentLabel := GetComponentLabel(tableChild.ComponentName)

						// 特殊处理：DDDateRangeField的label是数组格式
						if tableChild.ComponentName == "DDDateRangeField" && tableChildLabel == "" {
							if labelArr := GetPropStringArray(tableChild.Props, "label"); len(labelArr) > 0 {
								tableChildLabel = strings.Join(labelArr, "/")
							}
						}

						// 构建子字段说明
						var tableChildDesc strings.Builder
						if tableChildPlaceholder := GetPropString(tableChild.Props, "placeholder"); tableChildPlaceholder != "" {
							tableChildDesc.WriteString(fmt.Sprintf("提示: %s", tableChildPlaceholder))
						}
						// 子字段的选项
						if tableChildOptions, ok := tableChild.Props["options"].([]interface{}); ok && len(tableChildOptions) > 0 {
							if tableChildDesc.Len() > 0 {
								tableChildDesc.WriteString("; ")
							}
							tableChildDesc.WriteString("选项: ")
							opts := make([]string, 0)
							for _, opt := range tableChildOptions {
								if optMap, ok := opt.(map[string]interface{}); ok {
									if val, ok := optMap["value"].(string); ok {
										opts = append(opts, val)
									}
								}
							}
							tableChildDesc.WriteString(strings.Join(opts, ", "))
						}

						tableChildRequiredStr := "否"
						if tableChildRequired {
							tableChildRequiredStr = "✅ 是"
						}

						sb.WriteString(fmt.Sprintf("| %d.%d | └─ %s | %s | %s | %s |  |\n",
							i+1, subIdx, tableChildComponentLabel, tableChildLabel, tableChildRequiredStr, tableChildDesc.String()))
					}
				}
			}
		}
	}

	sb.WriteString("\n---\n\n")

	// 🔀 审批流程
	sb.WriteString("## 三、审批流程设计\n\n")

	// 检查是否有实际的审批节点
	if !hasApprovalNodes(&processNode) {
		sb.WriteString("⚠️ **注意：该表单无审批流程配置，仅有开始节点。**\n\n")
		sb.WriteString("建议检查原始数据或联系表单管理员确认流程配置。\n\n")
	} else {
		sb.WriteString("本审批流程采用**条件分支**模式，根据不同的业务场景自动路由到相应的审批链。\n\n")
		sb.WriteString("### 3.1 审批流程图\n\n")
		sb.WriteString("```\n")
		sb.WriteString(generateFlowChart(ctx, &processNode, 0))
		sb.WriteString("```\n\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*文档生成时间: %s*\n", time.Now().Format("2006-01-02")))

	// 写入文件
	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}