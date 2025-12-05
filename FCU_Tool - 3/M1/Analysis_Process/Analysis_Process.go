package Analysis_Process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/System_Analysis"
)

// 第一层固定：只分析 BuildDir/<Model>/simulink/systems/system_root.xml
func RunAnalysis(maxDepth int) {

	buildRoot := M1_Public_Data.BuildDir
	if buildRoot == "" {
		fmt.Println("❌ BuildDir 为空，请先调用 SetWorkDir() 初始化工作空间")
		return
	}

	// BuildDir 下的模型目录
	modelDirs, err := os.ReadDir(buildRoot)
	if err != nil {
		fmt.Println("❌ 无法读取 BuildDir 目录：", err)
		return
	}

	for _, modelEntry := range modelDirs {
		if !modelEntry.IsDir() {
			continue
		}

		modelName := modelEntry.Name()
		modelPath := filepath.Join(buildRoot, modelName)

		// 固定结构：<BuildDir>/<Model>/simulink/systems/system_root.xml
		sysDir := filepath.Join(modelPath, "simulink", "systems")
		xmlPath := filepath.Join(sysDir, "system_root.xml")

		if _, err := os.Stat(xmlPath); err != nil {
			continue // 模型没有 system_root.xml，跳过
		}

		fmt.Printf("🔍 分析模型 [%s] (最大深度: %d)\n", modelName, maxDepth)

		// 启动递归分析，从第1层开始，L1 没有父节点
		err = analyzeRecursive(sysDir, "system_root.xml", 1, maxDepth, "")
		if err != nil {
			fmt.Println("❌ 分析失败：", err)
			continue
		}
	}

	fmt.Printf("✅ 分析完成 (最大深度: %d)\n", maxDepth)
}

// 递归分析函数，根据 maxDepth 控制递归深度
// fatherName：当前这一层 System 对应的“父节点名称”，用于下一层输出 FatherNode 信息
func analyzeRecursive(dir, file string, currentLevel, maxDepth int, fatherName string) error {
	// 如果当前层数超过最大深度，停止递归
	if currentLevel > maxDepth {
		return nil
	}

	// 统一入口，由 System_Analysis 按 level 决定筛选逻辑
	subsystems, err := System_Analysis.AnalyzeSubSystemsInFile(dir, file, currentLevel, fatherName)
	if err != nil {
		return err
	}

	// 递归分析下一层
	if len(subsystems) > 0 && currentLevel < maxDepth {
		nextLevel := currentLevel + 1
		for _, sub := range subsystems {
			nextFile := fmt.Sprintf("system_%s.xml", sub.SID)
			nextFull := filepath.Join(dir, nextFile)

			if _, err := os.Stat(nextFull); err == nil {
				// 下一层的父节点 = 当前这一层的子系统名称
				nextFather := strings.TrimSpace(sub.Name)
				if err := analyzeRecursive(dir, nextFile, nextLevel, maxDepth, nextFather); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
