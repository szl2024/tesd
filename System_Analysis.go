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
// ✅ modelName：由上层传入（映射后的模型名），用于 txt 文件名
// fatherName：当前 system_xxx.xml 对应的父节点名称（L1 为空串）
func AnalyzeSubSystemsInFile(dir, file string, level int, modelName string, fatherName string) ([]SubSystemInfo, error) {
	switch level {
	case 1:
		return analyzeSubSystemsLevel1(dir, file, level, modelName, fatherName)
	case 2:
		return analyzeSubSystemsLevel2(dir, file, level, modelName, fatherName)
	case 3:
		return analyzeSubSystemsLevel3(dir, file, level, modelName, fatherName)
	default:
		return analyzeSubSystemsLevel3(dir, file, level, modelName, fatherName)
	}
}

// ======================== 逻辑 1（L1：过滤无效 SubSystem） ================================
func analyzeSubSystemsLevel1(dir, file string, level int, modelName string, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, modelName, true, fatherName)
}

// ======================== 逻辑 2（L2：不过滤 SubSystem） ================================
func analyzeSubSystemsLevel2(dir, file string, level int, modelName string, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, modelName, false, fatherName)
}

// ======================== 逻辑 3（L3+：非 Inport/Outport Block） =========================
func analyzeSubSystemsLevel3(dir, file string, level int, modelName string, fatherName string) ([]SubSystemInfo, error) {
	return analyzeNonPortBlocks(dir, file, level, modelName, fatherName)
}

// ======================== 通用 SubSystem 分析（移除递归，由外层控制） ====================
func analyzeSubSystemsCommon(dir, file string, level int, modelName string, applyLevel1Filter bool, fatherName string) ([]SubSystemInfo, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {
		if b.BlockType != "SubSystem" {
			continue
		}

		// level=1 时过滤：Ports 为空 / PortCounts 为空的 SubSystem 直接跳过
		if applyLevel1Filter && level == 1 {
			invalid := false

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

	// ✅ 关键：用“传入的 modelName”写 txt（这样就在生成 txt 时已完成改名）
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 分析失败 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}

// ======================== 非 Inport / Outport Block 分析（第 3 层及以后） ==================
func analyzeNonPortBlocks(dir, file string, level int, modelName string, fatherName string) ([]SubSystemInfo, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {
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

	// ✅ 同样用传入的 modelName
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 分析失败 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}
