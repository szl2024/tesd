package System_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/Port_Analysis"
)

// 用来保存 SubSystem 的 Name / SID / Level / BlockType
type SubSystemInfo struct {
	Name      string
	SID       string
	Level     int
	BlockType string
}

// P 标签
type xmlP struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

// PortCounts 标签
type xmlPortCounts struct {
	In      string `xml:"in,attr"`
	Out     string `xml:"out,attr"`
	Trigger string `xml:"trigger,attr"`
}

// Block
type xmlBlock struct {
	BlockType  string         `xml:"BlockType,attr"`
	Name       string         `xml:"Name,attr"`
	SID        string         `xml:"SID,attr"`
	PortCounts *xmlPortCounts `xml:"PortCounts"`
	Properties []xmlP         `xml:"P"`
}

type xmlSystem struct {
	Blocks []xmlBlock `xml:"Block"`
}

// ======================== 对外入口 ================================
// fatherName：当前 system_xxx.xml 对应的父节点名称（L1 为空串）
func AnalyzeSubSystemsInFile(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	switch level {
	case 1:
		return analyzeSubSystemsLevel1(dir, file, level, fatherName)
	case 2:
		return analyzeSubSystemsLevel2(dir, file, level, fatherName)
	case 3:
		return analyzeSubSystemsLevel3(dir, file, level, fatherName)
	default:
		// 第 3 层及以后统一按“非 Inport/Outport Block”处理
		return analyzeSubSystemsLevel3(dir, file, level, fatherName)
	}
}
//这里理应对L1和L2层进行两个函数分别分析，但是在分析函数中已经有分辨的了，所以L1和L2统一用相同的函数
// ======================== 逻辑 1（L1：过滤无效 SubSystem） ================================
func analyzeSubSystemsLevel1(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, true, fatherName)
}

// ======================== 逻辑 2（L2：不过滤 SubSystem） ================================
func analyzeSubSystemsLevel2(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, false, fatherName)
}

// ======================== 逻辑 3（L3+：非 Inport/Outport Block） =========================
func analyzeSubSystemsLevel3(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeNonPortBlocks(dir, file, level, fatherName)
}

// ======================== 通用 SubSystem 分析（移除递归，由外层控制） ====================
func analyzeSubSystemsCommon(dir, file string, level int, applyLevel1Filter bool, fatherName string) ([]SubSystemInfo, error) {
	
	//将要分析的文件的路径拼接
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	// 推出模型名：BuildDir/<Model>/simulink/systems → <Model>
	modelDir := filepath.Dir(filepath.Dir(dir))
	modelName := filepath.Base(modelDir)

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {

		if b.BlockType != "SubSystem" {
			continue
		}

		// === level=1 时过滤：Ports 为空 / PortCounts 为空的 SubSystem 直接跳过 ===
		if applyLevel1Filter && level == 1 {
			invalid := false
			// 当(1)和(2)中有其中一个不合符那就是不合格的block，即在第一层的时候，存在初始化的subsystem，需要进行过滤
			// (1) Ports = []
			for _, p := range b.Properties {
				if p.Name == "Ports" {
					v := strings.TrimSpace(p.Value)
					if v == "[]" || v == "" {
						invalid = true
						break
					}
				}
			}

			// (2) PortCounts 标签存在但为空
			if !invalid && b.PortCounts != nil {
				if b.PortCounts.In == "" && b.PortCounts.Out == "" && b.PortCounts.Trigger == "" {
					invalid = true
				}
			}

			if invalid {
				continue
			}
		}

		// 名字做一次规整，去掉换行、多空格
		rawName := strings.TrimSpace(b.Name)
		name := strings.Join(strings.Fields(rawName), " ")

		info := SubSystemInfo{
			Name:      name,
			SID:       b.SID,
			Level:     level,
			BlockType: b.BlockType, // "SubSystem"
		}
		result = append(result, info)
		blockSIDs = append(blockSIDs, b.SID)
	}

	// 把本层要输出的 BlockSID 列表交给 Port_Analysis，由它按 Block → Port 顺序统一输出
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 分析失败 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}

// ======================== 非 Inport / Outport Block 分析（第 3 层及以后） ==================
// 在指定 system_xxx.xml 中，找到所有 BlockType != "Inport" 且 != "Outport" 的 Block，
// 记录这些 Block 的 Name / BlockType / SID，并交给 Port_Analysis 做统一输出。
func analyzeNonPortBlocks(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	// 推出模型名：BuildDir/<Model>/simulink/systems → <Model>
	modelDir := filepath.Dir(filepath.Dir(dir))
	modelName := filepath.Base(modelDir)

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {

		// 跳过 Inport 和 Outport
		if b.BlockType == "Inport" || b.BlockType == "Outport" {
			continue
		}

		rawName := strings.TrimSpace(b.Name)
		name := strings.Join(strings.Fields(rawName), " ")

		info := SubSystemInfo{
			Name:      name,
			SID:       b.SID,
			Level:     level,
			BlockType: b.BlockType,
		}

		result = append(result, info)
		blockSIDs = append(blockSIDs, b.SID)
	}

	// 交给 Port_Analysis 做 Block + Port 的统一输出
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 分析失败 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}
