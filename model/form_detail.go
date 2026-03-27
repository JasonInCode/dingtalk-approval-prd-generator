// Package model 定义数据结构
package model

// FormDetailJSON 表单详情JSON结构
type FormDetailJSON struct {
	Success bool              `json:"success"`
	Data    FormDetailDataRaw `json:"data"`
}

// FormDetailDataRaw 原始数据
type FormDetailDataRaw struct {
	FormConfig     string        `json:"formConfig"`
	FormVo         FormVoRaw     `json:"formVo"`
	ProcessConfig  string        `json:"processConfig"`
	ProcessCode    string        `json:"processCode"`
	ProcessStatus  string        `json:"processStatus"`
	ProcessVersion string        `json:"processVersion"`
	ModifierName   string        `json:"modifierName"`
	ModifierTime   int64         `json:"modifierTime"`
}

// FormVoRaw 表单内容原始结构
type FormVoRaw struct {
	Content string `json:"content"`
}

// FormContent 解析后的表单内容
type FormContent struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Items       []PRDFormItem `json:"items"`
}

// PRDFormItem 表单项
type PRDFormItem struct {
	ComponentName string                 `json:"componentName"`
	Props         map[string]interface{} `json:"props"`
	Children      []PRDFormItem         `json:"children"` // 子字段（用于DDBizSuite等套件组件）
}

// PRDProcessNode 流程节点
type PRDProcessNode struct {
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	NodeId         string                 `json:"nodeId"`
	PrevId         string                 `json:"prevId"`
	Properties     map[string]interface{} `json:"properties"`
	ChildNode      *PRDProcessNode        `json:"childNode"`
	ConditionNodes []PRDProcessNode       `json:"conditionNodes"`
}

// FieldContext 字段上下文（用于并发安全的字段映射）
type FieldContext struct {
	OptionsMap   map[string]map[string]string // 字段选项映射 (fieldId -> {key: value})
	IdToLabelMap map[string]string            // 字段ID到名称映射
	OptionsList  map[string][]string          // 字段选项值列表映射 (fieldId -> [value1, value2, ...])，用于按索引匹配
}

// PRDExtraInfo PRD额外基本信息（来自审批列表）
type PRDExtraInfo struct {
	Category     string // 表单类型
	ModifiedTime string // 修改时间
	VisibleRange string // 可见范围
}

// 组件类型中文映射
var ComponentNameMap = map[string]string{
	"TextNote":              "📝 说明文字",
	"TextField":             "✏️ 单行文本",
	"TextareaField":         "📄 多行文本",
	"NumberField":           "🔢 数字输入",
	"MoneyField":            "💰 金额输入",
	"DDSelectField":         "📋 下拉选择",
	"DDMultiSelectField":    "📋 多选下拉",
	"DDPhotoField":          "📷 图片上传",
	"DDAttachment":          "📎 附件上传",
	"RelateField":           "🔗 关联表单",
	"CalculateField":        "🔢 计算公式",
	"InvoiceField":          "🧾 发票",
	"InnerContact":          "👤 内部联系人",
	"InnerContactField":     "👤 内部联系人",
	"DDDateField":           "📅 日期",
	"DDDateTimeField":       "📅 日期时间",
	"DDBizSuite":            "📦 业务套件",
	"DDHolidayField":        "🏖️ 请假组件",
	"DepartmentField":       "🏢 部门选择",
	"TableField":            "📊 明细表格",
	"FormRelateField":       "🔗 关联表单",
	"DDDateRangeField":      "📅 日期范围",
	"PhoneField":            "📱 电话",
	"IdCardField":           "🪪 身份证号",
	"TimeAndLocationField":  "📍 时间地点",
	"CommonField":           "📋 通用字段",
	"ExternalContactField":  "👥 外部联系人",
	"SignatureField":        "✍️ 签名",
	"CascadeField":          "📋 级联选择",
	"SeqNumberField":        "🔢 自动编号",
	"ColumnLayout":          "📐 分栏布局",
	"AddressField":          "📍 地址",
}