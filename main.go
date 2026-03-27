package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dingtalk-approval-prd-generator/api"
	"dingtalk-approval-prd-generator/model"
	"dingtalk-approval-prd-generator/output"
	"dingtalk-approval-prd-generator/utils"
)

const configFileName = "config.yaml"

func main() {
	// 命令行参数
	regenerateDir := flag.String("regenerate", "", "从本地JSON文件重新生成PRD文档，指定JSON文件所在目录")
	flag.Parse()

	// 如果指定了regenerate参数，则从本地JSON文件重新生成PRD
	if *regenerateDir != "" {
		regeneratePRD(*regenerateDir)
		return
	}

	fmt.Println("🚀 钉钉审批爬虫 & PRD 生成器")
	fmt.Println(strings.Repeat("=", 50))

	// 📁 加载配置文件
	config, err := LoadConfig(configFileName)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		fmt.Println("💡 请确保 config.yaml 文件存在且格式正确")
		return
	}

	// 🔍 检查 Cookie 是否配置
	if config.Cookie == "" {
		fmt.Println("❌ 请在 config.yaml 中配置 cookie")
		return
	}

	// 🔄 获取审批列表
	fmt.Println("\n🔄 正在获取审批列表...")
	apiResp, err := api.FetchApprovalList(config.Cookie)
	if err != nil {
		fmt.Printf("❌ 获取审批列表失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 成功获取 %s 审批列表\n", apiResp.Data.CorpName)

	// 📁 创建输出目录（使用公司名称）
	corpName := utils.SanitizeFilename(apiResp.Data.CorpName)
	outputDir := filepath.Join(config.OutputDir, "dingtalk_prd_"+corpName)
	// 检查目录是否存在，存在则删除
	if _, err := os.Stat(outputDir); err == nil {
		fmt.Printf("🗑️ 删除已存在的目录: %s\n", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			fmt.Printf("❌ 删除目录失败: %v\n", err)
			return
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ 创建输出目录失败: %v\n", err)
		return
	}
	fmt.Printf("📁 输出目录: %s\n", outputDir)

	// 📦 转换为结构化对象
	forms := api.ToApprovalForms(apiResp, utils.TimestampToDateTime)

	// 📝 输出 CSV 文件
	csvPath := filepath.Join(outputDir, "approval_forms.csv")
	if err := output.WriteCSV(forms, csvPath); err != nil {
		fmt.Printf("❌ 写入 CSV 失败: %v\n", err)
		return
	}
	fmt.Printf("✅ CSV 文件已生成: %s\n", csvPath)

	// 📊 打印摘要
	PrintSummary(forms)

	// 📋 并发获取表单详情并生成 PRD
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📝 开始生成 PRD 文档...")
	fmt.Printf("🔧 并发数: %d, 重试次数: %d\n", config.Workers, config.RetryCount)
	fmt.Println(strings.Repeat("=", 50))

	// 任务通道
	type Task struct {
		Index int
		Form  model.ApprovalForm
	}
	taskChan := make(chan Task, len(forms))

	// 生产任务
	for i, form := range forms {
		taskChan <- Task{Index: i + 1, Form: form}
	}
	close(taskChan)

	// 进度计数器
	var processedCount int32
	var successCount int32
	var failCount int32

	// 进度显示协程
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				processed := atomic.LoadInt32(&processedCount)
				success := atomic.LoadInt32(&successCount)
				fail := atomic.LoadInt32(&failCount)
				fmt.Printf("\r⏳ 进度: %d/%d | ✅ 成功: %d | ❌ 失败: %d", processed, len(forms), success, fail)
			case <-done:
				return
			}
		}
	}()

	// Worker 函数
	worker := func(workerId int) {
		for task := range taskChan {
			// 带重试的获取表单详情
			var detail *model.FormDetailJSON
			var err error
			for retry := 0; retry <= config.RetryCount; retry++ {
				detail, err = api.FetchFormDetail(config.Cookie, config.CsrfToken, config.CorpId, task.Form.Code)
				if err == nil && detail.Success {
					break
				}
				if retry < config.RetryCount {
					time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
				}
			}

			if err != nil || !detail.Success {
				atomic.AddInt32(&failCount, 1)
				atomic.AddInt32(&processedCount, 1)
				continue
			}

			// 生成安全的文件名
			safeTitle := utils.SanitizeFilename(task.Form.Title)
			if safeTitle == "" {
				safeTitle = task.Form.Code
			}
			safeCategory := utils.SanitizeFilename(task.Form.Category)
			if safeCategory == "" {
				safeCategory = "未分类"
			}

			// 文件名前缀：类型_名称
			filePrefix := fmt.Sprintf("%s_%s", safeCategory, safeTitle)

			// 💾 保存 JSON 文件
			jsonPath := filepath.Join(outputDir, fmt.Sprintf("%s_form.json", filePrefix))
			output.WriteJSON(detail, jsonPath)

			// 构建额外信息
			extraInfo := &model.PRDExtraInfo{
				Category:     task.Form.Category,
				ModifiedTime: task.Form.ModifiedTime,
				VisibleRange: task.Form.VisibleRange,
			}

			// 生成 PRD 文件名
			prdPath := filepath.Join(outputDir, fmt.Sprintf("%s_PRD.md", filePrefix))

			// 生成 PRD
			if err := output.GeneratePRD(detail, extraInfo, prdPath); err != nil {
				atomic.AddInt32(&failCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
			atomic.AddInt32(&processedCount, 1)
		}
	}

	// 启动 Worker
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(id)
		}(i)
	}

	// 等待所有任务完成
	wg.Wait()
	done <- true

	// 📊 输出统计
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 生成完成统计")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("  ✅ 成功: %d 个\n", successCount)
	fmt.Printf("  ❌ 失败: %d 个\n", failCount)
	fmt.Printf("  📁 输出目录: %s\n", outputDir)
}

// PrintSummary 打印数据摘要
func PrintSummary(forms []model.ApprovalForm) {
	fmt.Println("\n📊 数据统计:")
	fmt.Printf("  - 总记录数: %d\n", len(forms))

	// 按类型统计
	categoryCount := make(map[string]int)
	for _, f := range forms {
		categoryCount[f.Category]++
	}

	fmt.Println("\n📂 分类统计:")
	for cat, count := range categoryCount {
		fmt.Printf("  - %s: %d 条\n", cat, count)
	}
}

// regeneratePRD 从本地JSON文件重新生成PRD文档
func regeneratePRD(dir string) {
	fmt.Println("🔄 从本地JSON文件重新生成PRD文档")
	fmt.Printf("📁 目录: %s\n", dir)

	// 查找所有JSON文件
	files, err := filepath.Glob(filepath.Join(dir, "*_form.json"))
	if err != nil {
		fmt.Printf("❌ 查找JSON文件失败: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("❌ 未找到任何 *_form.json 文件")
		return
	}

	fmt.Printf("📋 找到 %d 个JSON文件\n", len(files))

	var successCount, failCount int32

	// 并发处理
	var wg sync.WaitGroup
	workers := 10
	taskChan := make(chan string, len(files))

	for _, f := range files {
		taskChan <- f
	}
	close(taskChan)

	// Worker函数
	worker := func() {
		defer wg.Done()
		for jsonPath := range taskChan {
			// 读取JSON文件
			detail, err := api.ParseFormDetail(jsonPath)
			if err != nil {
				fmt.Printf("❌ 解析失败: %s - %v\n", filepath.Base(jsonPath), err)
				atomic.AddInt32(&failCount, 1)
				continue
			}

			// 从JSON文件名推断PRD文件名
			base := strings.TrimSuffix(filepath.Base(jsonPath), "_form.json")
			prdPath := filepath.Join(dir, base+"_PRD.md")

			// 从CSV读取额外信息（如果存在）
			extraInfo := &model.PRDExtraInfo{
				Category:     "未知",
				ModifiedTime: utils.TimestampToDateTime(detail.Data.ModifierTime),
				VisibleRange: "未知",
			}

			// 尝试从文件名解析分类
			if idx := strings.Index(base, "_"); idx > 0 {
				extraInfo.Category = base[:idx]
			}

			// 生成PRD
			if err := output.GeneratePRD(detail, extraInfo, prdPath); err != nil {
				fmt.Printf("❌ 生成失败: %s - %v\n", base, err)
				atomic.AddInt32(&failCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}
	}

	// 启动workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}

	wg.Wait()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 重新生成完成")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("  ✅ 成功: %d 个\n", successCount)
	fmt.Printf("  ❌ 失败: %d 个\n", failCount)
}