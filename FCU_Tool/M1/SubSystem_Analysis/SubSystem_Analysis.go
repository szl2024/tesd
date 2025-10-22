package SubSystem_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/Port_Analysis"
)

// =============== 深度控制 ===============
const DefaultMaxDepth = 2 // 只要两层：顶层(1) + 直接子系统(2)

// 对外暴露的原函数签名，保持兼容；内部用带深度的版本
func AnalyzeSubSystems(modelName, sid string, parent *M1_Public_Data.SystemInfo) {
	AnalyzeSubSystemsWithDepth(modelName, sid, parent, 1, DefaultMaxDepth)
}

// 带深度控制的内部实现
func AnalyzeSubSystemsWithDepth(modelName, sid string, parent *M1_Public_Data.SystemInfo, depth, maxDepth int) {
	rootDir := M1_Public_Data.BuildDir
	xmlPath := filepath.Join(rootDir, modelName, "simulink", "systems", fmt.Sprintf("system_%s.xml", sid))

	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		fmt.Printf("⚠️ 未找到子系统文件：%s\n", xmlPath)
		return
	}

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		fmt.Printf("❌ 无法读取 %s：%v\n", xmlPath, err)
		return
	}

	var root XMLRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		fmt.Printf("❌ 无法解析 %s：%v\n", xmlPath, err)
		return
	}

	// ✅ 获取所有端口
	ports, _ := Port_Analysis.ParsePortsInXML(string(data))
	portMap := make(map[string]*M1_Public_Data.PortInfo)
	for _, p := range ports {
		portMap[p.SID] = p
	}

	// ✅ 建立连接关系
	connMap := parseConnections(string(data))

	// ✅ “穿透”中间计算模块
	connMap = flattenConnections(root.Blocks, connMap)

	for _, blk := range root.Blocks {
		if blk.BlockType != "SubSystem" {
			continue
		}
		if isPseudoSubSystem(blk.InnerXML) {
			fmt.Printf("⚠️ 跳过假性子系统：%s (SID=%s)\n", blk.Name, blk.SID)
			continue
		}

		// ========== 关键：深度限制（提前判断）==========
		// 顶层调用 depth=1；其直接子系统 depth=2。
		// 当 depth >= maxDepth 时，不再创建“更深一层”的子系统节点（即不产生第3层）。
		if depth >= maxDepth {
			continue
		}

		// ✅ 子系统继承父节点的 Model（此处是 depth+1 的那层）
		sub := &M1_Public_Data.SystemInfo{
			Model: parent.Model,
			Name:  blk.Name,
			SID:   blk.SID,
		}
		portAdded := make(map[string]bool)

		// 把与该子系统相连的端口挂上去（按连接关系 + portMap）
		for portSID, dstList := range connMap {
			for _, dstSID := range dstList {
				// 子系统 ↔ 端口 双向匹配
				if dstSID == blk.SID {
					if port, ok := portMap[portSID]; ok && !portAdded[port.SID] {
						sub.Port = append(sub.Port, port)
						portAdded[port.SID] = true
					}
				} else if portSID == blk.SID {
					if port, ok := portMap[dstSID]; ok && !portAdded[port.SID] {
						sub.Port = append(sub.Port, port)
						portAdded[port.SID] = true
					}
				}
			}
		}

		parent.SubSystem = append(parent.SubSystem, sub)

		// 到这里 depth 一定 < maxDepth，才会继续递归到下一层
		AnalyzeSubSystemsWithDepth(modelName, blk.SID, sub, depth+1, maxDepth)
	}
}

// =============== 以下为解析/工具，与之前版本一致 ===============

type XMLBlock struct {
	BlockType string `xml:"BlockType,attr"`
	Name      string `xml:"Name,attr"`
	SID       string `xml:"SID,attr"`
	InnerXML  string `xml:",innerxml"`
}

type XMLLine struct {
	P        []XMLParam  `xml:"P"`
	Branches []XMLBranch `xml:"Branch"`
}

type XMLBranch struct {
	P []XMLParam `xml:"P"`
}

type XMLParam struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

type XMLRoot struct {
	Blocks []XMLBlock `xml:"Block"`
	Lines  []XMLLine  `xml:"Line"`
}

// ✅ 解析连接（Src → 多个 Dst）
func parseConnections(xmlStr string) map[string][]string {
	result := make(map[string][]string)
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	var src string
	var dsts []string
	inLine := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if strings.EqualFold(se.Name.Local, "Line") {
				src = ""
				dsts = []string{}
				inLine = true
			} else if strings.EqualFold(se.Name.Local, "P") && inLine {
				var param XMLParam
				decoder.DecodeElement(&param, &se)
				if strings.EqualFold(param.Name, "Src") {
					src = strings.Split(param.Value, "#")[0]
				} else if strings.EqualFold(param.Name, "Dst") {
					dstSID := strings.Split(param.Value, "#")[0]
					if dstSID != "" {
						dsts = append(dsts, dstSID)
					}
				}
			} else if strings.EqualFold(se.Name.Local, "Branch") {
				var branch XMLBranch
				decoder.DecodeElement(&branch, &se)
				for _, p := range branch.P {
					if strings.EqualFold(p.Name, "Dst") {
						dstSID := strings.Split(p.Value, "#")[0]
						if dstSID != "" {
							dsts = append(dsts, dstSID)
						}
					}
				}
			}
		case xml.EndElement:
			if strings.EqualFold(se.Name.Local, "Line") {
				if src != "" && len(dsts) > 0 {
					result[src] = appendUnique(result[src], dsts)
					for _, d := range dsts {
						result[d] = appendUnique(result[d], []string{src})
					}
				}
				inLine = false
			}
		}
	}
	return result
}

// ✅ 去掉中间 Block，只保留 System ↔ Port 直接关系
func flattenConnections(blocks []XMLBlock, conn map[string][]string) map[string][]string {
	blockTypes := make(map[string]string)
	for _, blk := range blocks {
		blockTypes[blk.SID] = blk.BlockType
	}

	flattened := make(map[string][]string)
	for src, dsts := range conn {
		for _, dst := range dsts {
			srcType := blockTypes[src]
			dstType := blockTypes[dst]

			if isCalcBlock(srcType) && !isCalcBlock(dstType) {
				for upstream, targets := range conn {
					for _, t := range targets {
						if t == src {
							flattened[upstream] = appendUnique(flattened[upstream], []string{dst})
						}
					}
				}
			} else if !isCalcBlock(srcType) && isCalcBlock(dstType) {
				if nexts, ok := conn[dst]; ok {
					flattened[src] = appendUnique(flattened[src], nexts)
				}
			} else {
				flattened[src] = appendUnique(flattened[src], []string{dst})
			}
		}
	}
	return flattened
}

// ✅ 辅助函数：去重添加
func appendUnique(dst []string, items []string) []string {
	exists := make(map[string]bool)
	for _, d := range dst {
		exists[d] = true
	}
	for _, i := range items {
		if !exists[i] {
			dst = append(dst, i)
			exists[i] = true
		}
	}
	return dst
}

// ✅ 判断是否为计算块（需要穿透）
func isCalcBlock(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "operator") ||
		strings.Contains(t, "sum") ||
		strings.Contains(t, "logic") ||
		strings.Contains(t, "gain") ||
		strings.Contains(t, "math") ||
		strings.Contains(t, "delay")
}

// ✅ 判断假性子系统（无端口）
func isPseudoSubSystem(inner string) bool {
	s := strings.ReplaceAll(inner, " ", "")
	s = strings.ToLower(s)
	if strings.Contains(s, "<portcounts/>") ||
		strings.Contains(s, "<portcounts></portcounts>") {
		return true
	}
	if strings.Contains(s, `<pname="ports">[]</p>`) ||
		strings.Contains(s, `<pname="ports"></p>`) {
		return true
	}
	return false
}
