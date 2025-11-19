package Port_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/C_S_Analysis"
	"FCU_Tools/M1/Connection_Analysis"
	"FCU_Tools/M1/M1_Public_Data"
)

// 用来保存 Port 的信息
type PortInfo struct {
	Name      string
	SID       string
	Level     int
	BlockType string
	PortType  string
	Virtual   bool // true 表示伪 port（Block-Block 连接生成的虚拟端口）
}

// Block（这里只关心 BlockType / Name / SID）
type xmlBlock struct {
	BlockType string `xml:"BlockType,attr"`
	Name      string `xml:"Name,attr"`
	SID       string `xml:"SID,attr"`
}

type xmlSystem struct {
	Blocks []xmlBlock `xml:"Block"`
}

// 分析指定目录和文件的 Port + Block-Block 连接信息
// 输出格式：
// [Lx] Name: <BlockName>	BlockType=<BlockType>	SID=<SID> [FatherNode=xxx]
//     [Lx Port] Name: <真实Port名>	BlockType=<In/Outport>	SID=<SID> [PortType=S-R]
//     [Lx virtual Port] Name: <BlockA->BlockB[_n]>	BlockType=<In/Outport>	SID=<69->147> ...
//
// blockSIDs: 本层 System_Analysis 筛选出的 Block SID 列表，只对这些 Block 输出。
//            如果为空，则退回到“按 level 自动选择”的逻辑。
func AnalyzePortsInFile(dir, file string, level int, modelName, fatherName string, blockSIDs []string) error {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	// 1）建立 SID → Block 映射
	blocksBySID := make(map[string]xmlBlock)
	for _, b := range sys.Blocks {
		blocksBySID[b.SID] = b
	}

	// 2）决定本次要输出的 Block 集合：优先使用 blockSIDs；否则按 level 自动选择
	selected := make(map[string]struct{})
	var blockOrder []string

	if len(blockSIDs) > 0 {
		// 用 System_Analysis 已经筛好的 SID，保证和那边逻辑一致
		want := make(map[string]struct{})
		for _, sid := range blockSIDs {
			want[sid] = struct{}{}
		}
		for _, b := range sys.Blocks {
			if _, ok := want[b.SID]; ok {
				selected[b.SID] = struct{}{}
				blockOrder = append(blockOrder, b.SID)
			}
		}
	} else {
		// 兜底逻辑：如果没传 blockSIDs，根据 level 自己选
		for _, b := range sys.Blocks {
			if level == 1 || level == 2 {
				// 第 1、2 层：只分析 SubSystem
				if b.BlockType != "SubSystem" {
					continue
				}
			} else {
				// 第 3 层及以后：所有非 Inport / Outport 的 Block
				if b.BlockType == "Inport" || b.BlockType == "Outport" {
					continue
				}
			}
			selected[b.SID] = struct{}{}
			blockOrder = append(blockOrder, b.SID)
		}
	}

	// 3）收集所有真实 Port（Inport / Outport）
	portInfos := make(map[string]PortInfo)
	for _, b := range sys.Blocks {
		if b.BlockType != "Inport" && b.BlockType != "Outport" {
			continue
		}
		name := normalizeName(b.Name)
		portInfos[b.SID] = PortInfo{
			Name:      name,
			SID:       b.SID,
			Level:     level,
			BlockType: b.BlockType,
			PortType:  "S-R",
			Virtual:   false, // 真实端口
		}
	}

	// 4）用 Connection_Analysis 解析所有连接 Edge
	edges, err := Connection_Analysis.AnalyzeConnectionsInFile(dir, file)
	if err != nil {
		return fmt.Errorf("连接关系解析失败 [%s]: %w", fullPath, err)
	}

	// 4.1 先统计每一对 Block-SID 之间的连接条数，用于决定是否添加 _1/_2 后缀
	pairCounts := make(map[string]int) // "srcSID->dstSID" → 总条数
	for _, e := range edges {
		srcSID := e.SrcSID
		dstSID := e.DstSID

		// 只统计 Block-Block 的连接；只要双方都是合法 Block 即可
		if _, ok := blocksBySID[srcSID]; !ok {
			continue
		}
		if _, ok := blocksBySID[dstSID]; !ok {
			continue
		}

		// 排除端口本身（我们只想针对 Block-Block 多条线）
		if _, ok := portInfos[srcSID]; ok {
			continue
		}
		if _, ok := portInfos[dstSID]; ok {
			continue
		}

		baseKey := srcSID + "->" + dstSID
		pairCounts[baseKey]++
	}

	// 4.2 正式构建 Block → Ports 映射
	blockToPorts := make(map[string][]string)    // blockSID → []portSID
	seen := make(map[string]map[string]struct{}) // 去重用：blockSID → set(portSID)
	pairIndex := make(map[string]int)            // "srcSID->dstSID" → 当前第几条，用于 _1/_2

	for _, e := range edges {
		srcSID := e.SrcSID
		dstSID := e.DstSID

		_, srcIsPort := portInfos[srcSID]
		_, dstIsPort := portInfos[dstSID]

		_, srcIsSelectedBlock := selected[srcSID]
		_, dstIsSelectedBlock := selected[dstSID]

		// 情况 1：Src 是 Port，Dst 是关注的 Block（典型：Inport → SubSystem）
		if srcIsPort && dstIsSelectedBlock {
			if seen[dstSID] == nil {
				seen[dstSID] = make(map[string]struct{})
			}
			if _, ok := seen[dstSID][srcSID]; !ok {
				blockToPorts[dstSID] = append(blockToPorts[dstSID], srcSID)
				seen[dstSID][srcSID] = struct{}{}
			}
		}

		// 情况 2：Dst 是 Port，Src 是关注的 Block（典型：Block → Outport）
		if dstIsPort && srcIsSelectedBlock {
			if seen[srcSID] == nil {
				seen[srcSID] = make(map[string]struct{})
			}
			if _, ok := seen[srcSID][dstSID]; !ok {
				blockToPorts[srcSID] = append(blockToPorts[srcSID], dstSID)
				seen[srcSID][dstSID] = struct{}{}
			}
		}

		// 情况 3：Block 和 Block 直接相连（例如 66#out:1 → 69#in:3）
		// 只要一端是“关注的 Block”，就要生成对应的虚拟 port
		if !srcIsPort && !dstIsPort {
			srcBlk, ok1 := blocksBySID[srcSID]
			dstBlk, ok2 := blocksBySID[dstSID]
			if !ok1 || !ok2 {
				continue
			}

			baseKey := srcSID + "->" + dstSID
			total := pairCounts[baseKey]
			if total == 0 {
				total = 1
			}
			pairIndex[baseKey]++
			idx := pairIndex[baseKey]

			srcName := normalizeName(srcBlk.Name)
			dstName := normalizeName(dstBlk.Name)

			// 生成连接名字：多条线则带 _1/_2 后缀
			label := ""
			if total > 1 {
				label = fmt.Sprintf("%s->%s_%d", srcName, dstName, idx)
			} else {
				label = fmt.Sprintf("%s->%s", srcName, dstName)
			}

			// 显示用 SID：只保留 "srcSID->dstSID"
			displaySID := fmt.Sprintf("%s->%s", srcSID, dstSID)

			// 3.1 如果 Src 是关注的 Block：在 Src 下面挂虚拟 Outport
			if srcIsSelectedBlock {
				if seen[srcSID] == nil {
					seen[srcSID] = make(map[string]struct{})
				}

				virtKey := fmt.Sprintf("%s_OUT_%d", baseKey, idx)
				if _, ok := portInfos[virtKey]; !ok {
					portInfos[virtKey] = PortInfo{
						Name:      label,
						SID:       displaySID,
						Level:     level,
						BlockType: "Outport",
						PortType:  "S-R",
						Virtual:   true,
					}
				}

				if _, ok := seen[srcSID][virtKey]; !ok {
					blockToPorts[srcSID] = append(blockToPorts[srcSID], virtKey)
					seen[srcSID][virtKey] = struct{}{}
				}
			}

			// 3.2 如果 Dst 是关注的 Block：在 Dst 下面挂虚拟 Inport
			if dstIsSelectedBlock {
				if seen[dstSID] == nil {
					seen[dstSID] = make(map[string]struct{})
				}

				virtKey := fmt.Sprintf("%s_IN_%d", baseKey, idx)
				if _, ok := portInfos[virtKey]; !ok {
					portInfos[virtKey] = PortInfo{
						Name:      label,
						SID:       displaySID,
						Level:     level,
						BlockType: "Inport",
						PortType:  "S-R",
						Virtual:   true,
					}
				}

				if _, ok := seen[dstSID][virtKey]; !ok {
					blockToPorts[dstSID] = append(blockToPorts[dstSID], virtKey)
					seen[dstSID][virtKey] = struct{}{}
				}
			}
		}
	}

	// 5）统一按 “Block → Ports” 顺序输出到 txt
	if M1_Public_Data.TxtDir == "" || modelName == "" {
		// 没有输出目录就直接结束
		return nil
	}

	txtPath := filepath.Join(M1_Public_Data.TxtDir, modelName+".txt")
	f, err := os.OpenFile(txtPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("无法写入 txt 文件 [%s]: %w", txtPath, err)
	}
	defer f.Close()

	for _, sid := range blockOrder {
		blk, ok := blocksBySID[sid]
		if !ok {
			continue
		}

		name := normalizeName(blk.Name)

		// 先输出 Block 自身信息，带上 FatherNode（从第二层开始）
		var blockLine string
		if fatherName != "" && level >= 2 {
			blockLine = fmt.Sprintf(
				"[L%d] Name: %-10s\tBlockType=%-10s\tSID=%-10s\tFatherNode=%-10s\n",
				level, name, blk.BlockType, blk.SID, fatherName,
			)
		} else {
			blockLine = fmt.Sprintf(
				"[L%d] Name: %s\tBlockType=%s\tSID=%s\n",
				level, name, blk.BlockType, blk.SID,
			)
		}

		if _, err := f.WriteString(blockLine); err != nil {
			return err
		}

		// 再输出这个 Block 的所有 Port / 伪 Port 信息
		if ports, ok := blockToPorts[sid]; ok {
			for _, psid := range ports {
				pinfo, ok := portInfos[psid]
				if !ok {
					continue
				}

				// 根据是否虚拟端口，选择不同标签
				label := "Port"
				if pinfo.Virtual {
					label = "virtual Port"
				}

				// L1 才输出 PortType；L2 及以后不输出 PortType
				var portLine string
				if level == 1 {
					portLine = fmt.Sprintf(
						"\t[L%d %s] Name: %-40s\tBlockType=%-10s\tSID=%-10s\tPortType=%-10s\n",
						level, label, pinfo.Name, pinfo.BlockType, pinfo.SID, pinfo.PortType,
					)
				} else {
					portLine = fmt.Sprintf(
						"\t[L%d %s] Name:%-40s\tBlockType=%-10s\tSID=%-10s\n",
						level, label, pinfo.Name, pinfo.BlockType, pinfo.SID,
					)
				}

				if _, err := f.WriteString(portLine); err != nil {
					return err
				}
			}
		}
	}

	// 6）在 L1 追加 C-S 端口（来自 BuildDir\<Model>\simulink\graphicalInterface.xml）
	if level == 1 {
		csPorts, err := C_S_Analysis.GetCSPorts(modelName)
		if err != nil {
			// 不打断整体流程，只提示一下
			fmt.Printf("⚠️ 解析 C-S 端口失败：%v\n", err)
		} else if len(csPorts) > 0 {
			for _, p := range csPorts {
				line := fmt.Sprintf(
					"\t[L1 Port] Name: %-40s\tBlockType=%-10s\tSID=%-10s\tPortType=%-10s\n",
					p.Name, p.BlockType, p.SID, p.PortType,
				)
				if _, err := f.WriteString(line); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// 把名字里的换行 / 多余空白压成一个空格
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}
