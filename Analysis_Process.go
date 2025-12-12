package Analysis_Process

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/System_Analysis"
	"FCU_Tools/Public_data"
)

// 从 asw.csv 中构建 runnable → 模型名 映射
// 约定：第 4 列(index 3)=模型名；第 6 列(index 5)=runnable 名
func buildRunnableToModelMap() (map[string]string, error) {
	result := make(map[string]string)

	csvPath := Public_data.ConnectorFilePath
	if csvPath == "" {
		return result, nil
	}

	f, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("打开 asw.csv 失败（ConnectorFilePath=%s）: %v", csvPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取 asw.csv 内容失败: %v", err)
	}

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) <= 5 {
			continue
		}
		modelName := strings.TrimSpace(row[3])
		runnable := strings.TrimSpace(row[5])
		if modelName == "" || runnable == "" {
			continue
		}
		result[runnable] = modelName
	}

	return result, nil
}

// 把 BuildDir 下的目录名（通常是 runnable）映射成模型名；找不到映射就原样返回
func mapModelNameByRunnable(runnable string, runnableToModel map[string]string) string {
	if runnableToModel == nil {
		return runnable
	}
	if v, ok := runnableToModel[runnable]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return runnable
}

// 第一层固定：只分析 BuildDir/<Model>/simulink/systems/system_root.xml
func RunAnalysis(maxDepth int) {
	buildRoot := M1_Public_Data.BuildDir
	if buildRoot == "" {
		fmt.Println("❌ BuildDir 为空，请先调用 SetWorkDir() 初始化工作空间")
		return
	}

	// ✅ 关键：在写 txt 前就准备好 runnable→model 映射
	runnableToModel, err := buildRunnableToModelMap()
	if err != nil {
		fmt.Println("❌ 读取 asw.csv 失败：", err)
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

		// BuildDir 的目录名通常是 runnable（或 slx 解压后的名字）
		runnableName := modelEntry.Name()
		// ✅ 用 asw.csv 映射后的“模型名”作为 txt 文件名/输出名
		modelName := mapModelNameByRunnable(runnableName, runnableToModel)

		modelPath := filepath.Join(buildRoot, runnableName)

		// 固定结构：<BuildDir>/<Model>/simulink/systems/system_root.xml
		sysDir := filepath.Join(modelPath, "simulink", "systems")
		xmlPath := filepath.Join(sysDir, "system_root.xml")

		if _, err := os.Stat(xmlPath); err != nil {
			continue // 模型没有 system_root.xml，跳过
		}

		fmt.Printf("🔍 分析模型 [%s] (最大深度: %d)\n", modelName, maxDepth)

		// 启动递归分析，从第1层开始，L1 没有父节点
		err = analyzeRecursive(sysDir, "system_root.xml", 1, maxDepth, modelName, "")
		if err != nil {
			fmt.Println("❌ 分析失败：", err)
			continue
		}
	}

	fmt.Printf("✅ 分析完成 (最大深度: %d)\n", maxDepth)
}

// 递归分析函数，根据 maxDepth 控制递归深度
// modelName：用于 txt 文件名（要用映射后的模型名）
// fatherName：当前这一层 System 对应的“父节点名称”，用于下一层输出 FatherNode 信息
func analyzeRecursive(dir, file string, currentLevel, maxDepth int, modelName string, fatherName string) error {
	if currentLevel > maxDepth {
		return nil
	}

	// ✅ 关键：把 modelName 直接传给 System_Analysis，别让它自己从路径推 runnable
	subsystems, err := System_Analysis.AnalyzeSubSystemsInFile(dir, file, currentLevel, modelName, fatherName)
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
				nextFather := strings.TrimSpace(sub.Name)
				if err := analyzeRecursive(dir, nextFile, nextLevel, maxDepth, modelName, nextFather); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
