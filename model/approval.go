// Package model 定义数据结构
package model

// ApiResponse API 响应结构
type ApiResponse struct {
	Data       Data   `json:"data"`
	ErrorCode  string `json:"errorCode"`
	ErrorMsg   string `json:"errorMessage"`
	HttpStatus string `json:"httpStatus"`
	Success    bool   `json:"success"`
}

// Data 数据主体
type Data struct {
	CorpName             string           `json:"corpName"`
	SortedDirProcessList []DirProcessList `json:"sortedDirProcessList"`
	Open                 bool             `json:"open"`
}

// DirProcessList 审批分类目录
type DirProcessList struct {
	DirName                    string             `json:"dirName"`
	DirId                      string             `json:"dirId"`
	SortedProcessAndFormVoList []ProcessAndFormVo `json:"sortedProcessAndFormVoList"`
}

// ProcessAndFormVo 审批流程表单
type ProcessAndFormVo struct {
	FlowTitle          string `json:"flowTitle"`
	ProcessCode        string `json:"processCode"`
	ProcessId          int64  `json:"processId"`
	Description        string `json:"description"`
	ProcessStatus      string `json:"processStatus"`
	ModifierNick       string `json:"modifierNick"`
	GmtModified        int64  `json:"gmtModified"`
	VisibleSummaryText string `json:"visibleSummaryText"`
	OriginatorId       string `json:"originatorId"`
	Modifier           string `json:"modifier"`
	IconRealUrl        string `json:"iconRealUrl"`
	IconUrl            string `json:"iconUrl"`
	FormVo             FormVo `json:"formVo"`
}

// FormVo 表单信息
type FormVo struct {
	Memo        string `json:"memo"`
	PushSwitch  string `json:"pushSwitch"`
	GmtModified int64  `json:"gmtModified"`
}

// ApprovalForm 审批表单（用于 CSV 输出的结构化对象）
type ApprovalForm struct {
	Category     string // 类型
	Title        string // 流程标题
	Code         string // 流程编码
	Description  string // 描述
	Status       string // 状态
	ModifierNick string // 修改者昵称
	ModifiedTime string // 修改时间（格式化后）
	VisibleRange string // 可见范围描述
}