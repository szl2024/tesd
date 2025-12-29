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

// Port 정보를 저장하는 데 사용됩니다.
type PortInfo struct {
	Name      string
	SID       string
	Level     int
	BlockType string
	PortType  string
	Virtual   bool // true는 의사 포트(블록-블록 연결로 생성된 가상 포트)를 의미합니다.
}

// 블록(여기서는 BlockType / Name / SID만 관심)
type xmlBlock struct {
	BlockType string `xml:"BlockType,attr"`
	Name      string `xml:"Name,attr"`
	SID       string `xml:"SID,attr"`
}

type xmlSystem struct {
	Blocks []xmlBlock `xml:"Block"`
}

// 지정된 디렉터리와 파일의 Port + 블록-블록 연결 정보를 분석합니다.
// 출력 형식：
// [Lx] Name: <BlockName>	BlockType=<BlockType>	SID=<SID> [FatherNode=xxx]
//     [Lx Port] Name: <真实Port名>	BlockType=<In/Outport>	SID=<SID> [PortType=S-R]
//     [Lx virtual Port] Name: <BlockA->BlockB[_n]>	BlockType=<In/Outport>	SID=<69->147> ...
//
// blockSIDs: 본 레이어의 System_Analysis에서 필터링된 Block SID 목록이며, 이 Block들에 대해서만 출력합니다.
//            비어 있으면 “level에 따라 자동 선택” 로직으로 되돌아갑니다.
func AnalyzePortsInFile(dir, file string, level int, modelName, fatherName string, blockSIDs []string) error {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("XML 읽기 실패 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return fmt.Errorf("XML 파싱 실패 [%s]: %w", fullPath, err)
	}

	// 1）SID → Block 매핑 구축
	blocksBySID := make(map[string]xmlBlock)
	for _, b := range sys.Blocks {
		blocksBySID[b.SID] = b
	}

	// 2）이번에 출력할 Block 집합을 결정합니다: blockSIDs를 우선 사용하고, 그렇지 않으면 level에 따라 자동 선택합니다.
	selected := make(map[string]struct{})
	var blockOrder []string

	if len(blockSIDs) > 0 {
		// System_Analysis에서 이미 선별한 SID를 사용하여, 그쪽 로직과 일치하도록 보장합니다.
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
		// 대체 로직: blockSIDs가 전달되지 않으면 level에 따라 직접 선택합니다.
		for _, b := range sys.Blocks {
			if level == 1 || level == 2 {
				// 1, 2층: SubSystem만 분석합니다.
				if b.BlockType != "SubSystem" {
					continue
				}
			} else {
				// 3층 및 이후: Inport/Outport가 아닌 모든 Block
				if b.BlockType == "Inport" || b.BlockType == "Outport" {
					continue
				}
			}
			selected[b.SID] = struct{}{}
			blockOrder = append(blockOrder, b.SID)
		}
	}

	// 3）모든 실제 Port(Inport/Outport)를 수집합니다.
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
			Virtual:   false, // 실제 포트
		}
	}

	// 4）Connection_Analysis로 모든 연결 Edge를 파싱합니다.
	edges, err := Connection_Analysis.AnalyzeConnectionsInFile(dir, file)
	if err != nil {
		return fmt.Errorf("连接关系解析失败 [%s]: %w", fullPath, err)
	}

	// 4.1 먼저 각 Block-SID 쌍 사이의 연결 개수를 집계하여, _1/_2 접미사를 추가할지 결정하는 데 사용합니다.
	pairCounts := make(map[string]int) // srcSID->dstSID → 총 개수
	for _, e := range edges {
		srcSID := e.SrcSID
		dstSID := e.DstSID

		// Block-Block 연결만 집계하며, 양쪽 모두 유효한 Block이면 됩니다.
		if _, ok := blocksBySID[srcSID]; !ok {
			continue
		}
		if _, ok := blocksBySID[dstSID]; !ok {
			continue
		}

		// 포트 자체는 제외합니다(우리는 Block-Block 다중 연결만 대상으로 합니다).
		if _, ok := portInfos[srcSID]; ok {
			continue
		}
		if _, ok := portInfos[dstSID]; ok {
			continue
		}

		baseKey := srcSID + "->" + dstSID
		pairCounts[baseKey]++
	}

	// 4.2 정식으로 Block → Ports 매핑을 구축합니다.
	blockToPorts := make(map[string][]string)    // blockSID → []portSID
	seen := make(map[string]map[string]struct{}) // 중복 제거에 사용: blockSID → set(portSID)
	pairIndex := make(map[string]int)            // srcSID->dstSID → 현재 몇 번째인지, _1/_2에 사용합니다.

	for _, e := range edges {
		srcSID := e.SrcSID
		dstSID := e.DstSID

		_, srcIsPort := portInfos[srcSID]
		_, dstIsPort := portInfos[dstSID]

		_, srcIsSelectedBlock := selected[srcSID]
		_, dstIsSelectedBlock := selected[dstSID]

		// 상황 1: Src가 Port이고 Dst가 관심 Block인 경우(전형적: Inport → SubSystem)
		if srcIsPort && dstIsSelectedBlock {
			if seen[dstSID] == nil {
				seen[dstSID] = make(map[string]struct{})
			}
			if _, ok := seen[dstSID][srcSID]; !ok {
				blockToPorts[dstSID] = append(blockToPorts[dstSID], srcSID)
				seen[dstSID][srcSID] = struct{}{}
			}
		}

		// 상황 2: Dst가 Port이고 Src가 관심 Block인 경우(전형적: Block → Outport)
		if dstIsPort && srcIsSelectedBlock {
			if seen[srcSID] == nil {
				seen[srcSID] = make(map[string]struct{})
			}
			if _, ok := seen[srcSID][dstSID]; !ok {
				blockToPorts[srcSID] = append(blockToPorts[srcSID], dstSID)
				seen[srcSID][dstSID] = struct{}{}
			}
		}

		// 상황 3: Block과 Block이 직접 연결된 경우(예: 66#out:1 → 69#in:3)
		// 한쪽이라도 “관심 Block”이면 해당 가상 포트를 생성해야 합니다.
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

			// 연결 이름 생성: 여러 선이면 _1/_2 접미사를 붙입니다.
			label := ""
			if total > 1 {
				label = fmt.Sprintf("%s->%s_%d", srcName, dstName, idx)
			} else {
				label = fmt.Sprintf("%s->%s", srcName, dstName)
			}

			// 표시용 SID: "srcSID->dstSID"만 유지합니다.
			displaySID := fmt.Sprintf("%s->%s", srcSID, dstSID)

			// 3.1 Src가 관심 Block이면: Src 아래에 가상 Outport를 추가합니다.
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

			// 3.2 Dst가 관심 Block이면: Dst 아래에 가상 Inport를 추가합니다.
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

	// 5）통일하여 “Block → Ports” 순서로 txt에 출력합니다.
	if M1_Public_Data.TxtDir == "" || modelName == "" {
		// 출력 디렉터리가 없으면 바로 종료합니다.
		return nil
	}

	txtPath := filepath.Join(M1_Public_Data.TxtDir, modelName+".txt")
	f, err := os.OpenFile(txtPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("txt 파일에 쓸 수 없습니다. [%s]: %w", txtPath, err)
	}
	defer f.Close()

	for _, sid := range blockOrder {
		blk, ok := blocksBySID[sid]
		if !ok {
			continue
		}

		name := normalizeName(blk.Name)

		// 먼저 Block 자체 정보를 출력하고, FatherNode를 함께 표시합니다(2층부터).
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

		// 그다음 이 Block의 모든 Port/의사 포트 정보를 출력합니다.
		if ports, ok := blockToPorts[sid]; ok {
			for _, psid := range ports {
				pinfo, ok := portInfos[psid]
				if !ok {
					continue
				}

				// 가상 포트 여부에 따라 다른 태그를 선택합니다.
				label := "Port"
				if pinfo.Virtual {
					label = "virtual Port"
				}

				// L1에서만 PortType을 출력하고, L2 및 이후에는 PortType을 출력하지 않습니다.
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

	// 6）L1에서 C-S 포트를 추가합니다(BuildDir\<Model>\simulink\graphicalInterface.xml에서 가져옴).
	if level == 1 {
		csPorts, err := C_S_Analysis.GetCSPorts(modelName)
		if err != nil {
			// 전체 흐름을 중단하지 않고, 안내만 합니다.
			fmt.Printf("⚠️ C-S 포트 파싱 실패：%v\n", err)
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

// 이름에 있는 줄바꿈/불필요한 공백을 하나의 공백으로 압축합니다.
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}
