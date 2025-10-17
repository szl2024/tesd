package Subsystem_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/Port_Analysis"
	"bufio"
	"FCU_Tools/M1/Connect_Analysis"
)
const portNameWidth = 35
const blockNameWidth = 35
const blockInfoWidth = 40 //Control the alignment of the length within parentheses

type (
	// Analyze the contextual state
	parserState struct {
		currentBlock     blockState
		currentLine      lineState
		currentSystem    *M1_Public_Data.System
		elementStack     []xml.StartElement
		currentPName     string
		currentPContent  string
		hasPortAttr      bool 
		portsFromList []int
	}
	

	blockState struct {
		Type        string
		Name        string
		SID         string
		IsAtomic    bool
		SystemRef   string
		PortCounts  portCounts
	}

	lineState struct {
		Src      string
		Dst      string
		Branches []string  // Branch connection target
	}
	

	portCounts struct {
		In      int
		Out     int
		Trigger int
	}
)

// AnalyzeSubSystemXML는 rootSystem을 루트로 하여 큐 기반의 너비 우선 방식으로
// 하위 시스템 XML을 하나씩 파싱한다.
// 시스템 계층 및 포트/연결 정보를 구성하고,
// M1_Public_Data.TxtDir에 전체 모델의 구조와 통계 TXT를 출력한다.
//
// 프로세스:
//   1) 큐 초기화: 루트 (xmlPath, rootSystem)를 큐에 삽입.
//   2) 큐에서 반복적으로 꺼내며 processSystemFile을 호출하여 단일 시스템을 파싱,
//      이 과정에서 참조된 SystemRef에 대응하는 XML을 큐에 추가할 수 있음.
//   3) 전체 파싱 완료 후, printSystemInfoToWriter를 호출하여 계층 및 통계 정보를
//      builder에 연결하고 <modelName>.txt로 기록.

func AnalyzeSubSystemXML(xmlPath string, rootSystem *M1_Public_Data.System, modelName string) error{
	queue := []struct {
		path string
		sys  *M1_Public_Data.System
	}{{xmlPath, rootSystem}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if err := processSystemFile(item.path, item.sys, &queue); err != nil {
			return err
		}
	}
	
	outputPath := filepath.Join(M1_Public_Data.TxtDir, modelName+".txt")

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📦 The results of the system architecture analysis are as follows：\n"))
	printSystemInfoToWriter(rootSystem, rootSystem, "", &builder)

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		return err
	}
	defer func() {
    	if cerr := f.Close(); cerr != nil {
        	fmt.Printf("❌ Failed to close file: %v\n", cerr)
    	}
	}()

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	_, err = writer.WriteString(builder.String())
	if err != nil {
		fmt.Printf("❌ Failed to write model structure: %v\n", err)
		return err
	}
	//fmt.Println("\n=== System statistics ===")
	return nil
}

// processSystemFile은 단일 시스템 XML(path 경로)을 파싱하여
// 블록, 포트, 연결 및 하위 시스템 참조를 system 객체로 변환한다.
//
// 파싱 요점:
//   - Block: Inport/Outport → Port_Analysis.PortAnalysis를 호출하여 포트 추가;
//            SubSystem → 하위 System 생성 후 SystemRef에 따라 큐에 삽입;
//            그 외 → 일반 기능 블록으로 기록.
//   - Line: Src/Dst와 Branches를 파싱하고,
//           Line 종료 시 Connect_Analysis.ConnectAnalysis를 호출하여 연결(분기 포함)을 기록.
//   - P 태그: TreatAsAtomicUnit/Port/Ports/Src/Dst 등을 처리하여
//             현재 블록/라인 상태를 유지.

func processSystemFile(path string, system *M1_Public_Data.System, queue *[]struct{path string; sys *M1_Public_Data.System}) error {
	
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Fail to open file: %w", err)
	}
	defer file.Close()

	state := &parserState{
		currentSystem: system,
	}

	decoder := xml.NewDecoder(file)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch elem := token.(type) {
		case xml.StartElement:
			state.handleStartElement(elem)
			
		case xml.EndElement:
			state.handleEndElement(elem, queue, path)
			
		case xml.CharData:
			state.handleCharData(elem)
		}
	}
	return nil
}

// handleStartElement Start 태그 처리:
//   요소 스택을 유지하고,
//   태그 유형에 따라 initBlockState/parsePortCounts/initPState/parseSystemRef 등에 분배한다.


func (s *parserState) handleStartElement(elem xml.StartElement) {
	s.elementStack = append(s.elementStack, elem)
	
	switch elem.Name.Local {
	case "Block":
		s.initBlockState(elem)
	case "Line":
		s.currentLine = lineState{}
	case "PortCounts":
		s.parsePortCounts(elem)
	case "P":
		s.initPState(elem)
	case "System":
		s.parseSystemRef(elem)
	}
}

// handleEndElement End 태그 처리:
//   스택에서 팝(pop);
//   Block/Line/P 종료 시 각각 processBlock/processLine/processPContent를 호출한다.


func (s *parserState) handleEndElement(elem xml.EndElement, queue *[]struct{path string; sys *M1_Public_Data.System}, currentPath string) {
	if len(s.elementStack) > 0 {
		s.elementStack = s.elementStack[:len(s.elementStack)-1]
	}

	switch elem.Name.Local {
	case "Block":
		s.processBlock(queue, currentPath)
	case "Line":
		s.processLine()
	case "P":
		s.processPContent()
	}
}

// handleCharData 문자 데이터 처리: 현재 P 태그의 내용 텍스트를 캐시(공백 제거).

func (s *parserState) handleCharData(data xml.CharData) {
	s.currentPContent = strings.TrimSpace(string(data))
}

// initBlockState 현재 Block 상태(이름, SID, 유형)를 초기화하고,
// 포트 속성과 관련된 임시 상태를 초기화한다.

func (s *parserState) initBlockState(elem xml.StartElement) {
	s.currentBlock = blockState{}
	s.hasPortAttr = false
	s.portsFromList = nil 
	for _, attr := range elem.Attr {
		switch attr.Name.Local {
		case "Name":
			s.currentBlock.Name = attr.Value
		case "SID":
    		s.currentBlock.SID = attr.Value
		case "BlockType":
			s.currentBlock.Type = attr.Value
		}
	}
}


// parsePortCounts PortCounts 노드의 in/out/trigger 개수를 파싱하여
// 현재 Block의 포트 개수로 기록한다.

func (s *parserState) parsePortCounts(elem xml.StartElement) {
	for _, attr := range elem.Attr {
		switch attr.Name.Local {
		case "in":
			s.currentBlock.PortCounts.In, _ = strconv.Atoi(attr.Value)
		case "out":
			s.currentBlock.PortCounts.Out, _ = strconv.Atoi(attr.Value)
		case "trigger":
			s.currentBlock.PortCounts.Trigger, _ = strconv.Atoi(attr.Value)
		}
	}
}

// initPState 현재 P 태그 상태를 초기화
// (Name 속성을 기록하고, 내용 캐시를 초기화한다).

func (s *parserState) initPState(elem xml.StartElement) {
	s.currentPContent = ""
	for _, attr := range elem.Attr {
		if attr.Name.Local == "Name" {
			s.currentPName = attr.Value
		}
	}
}

// parseSystemRef System 태그의 Ref 속성을 파싱하여
// 현재 Block의 SystemRef에 기록한다.

func (s *parserState) parseSystemRef(elem xml.StartElement) {
	for _, attr := range elem.Attr {
		if attr.Name.Local == "Ref" {
			s.currentBlock.SystemRef = attr.Value
		}
	}
}

// processBlock 현재 Block.Type에 따라 분기:
//   - Inport/Outport → 포트 생성.
//   - SubSystem → 하위 System 생성, class/system 구분(TreatAsAtomicUnit으로 결정),
//                 SystemRef가 있으면 해당 XML을 큐에 추가하고 부모 시스템에 연결.
//   - 기타 → 일반 기능 블록으로 시스템에 추가.
// 주의: 여기서는 파일을 기록하지 않고, 메모리 구조와 이후 파싱 대기 큐만 구성한다.


func (s *parserState) processBlock(queue *[]struct{ path string; sys *M1_Public_Data.System }, currentPath string) {
	switch s.currentBlock.Type {
	case "Inport", "Outport":
		Port_Analysis.PortAnalysis(
			s.currentBlock.Name,
			s.currentBlock.SID,
			s.currentBlock.Type,
			s.hasPortAttr, 
			s.currentSystem,
		)

	case "SubSystem":
		// Create subsystem
		subSystem := M1_Public_Data.NewSystemFromBlock(
			s.currentBlock.Name,
			s.currentBlock.SID,
			s.currentBlock.PortCounts.In,
			s.currentBlock.PortCounts.Out,
			true,   
		)
		

		// No longer distinguish between System and Class
subSystem.Type = "subsystem"


		// If there is a system reference, recursively read its XML file
		if s.currentBlock.SystemRef != "" {
			newPath := generateNewPath(currentPath, s.currentBlock.SystemRef)
			*queue = append(*queue, struct {
				path string
				sys  *M1_Public_Data.System
			}{
				path: newPath,
				sys:  subSystem,
			})
		}

		// Add SubSystem to the System list of the parent system
		s.currentSystem.System = append(s.currentSystem.System, subSystem)

	default:
		// Common functional blocks, such as Terminator, Gain, etc
		s.currentSystem.Block = append(s.currentSystem.Block, &M1_Public_Data.Block{
			Name: strings.TrimSpace(s.currentBlock.Name),
			SID:  s.currentBlock.SID,
			Type: s.currentBlock.Type,
		})
	}
}


// processLine 한 Line이 끝날 때,
//   파싱된 Src → Dst 및 모든 Branch 연결을 currentSystem에 등록한다
//   (Connect_Analysis.ConnectAnalysis 호출).


func (s *parserState) processLine() {
	if s.currentLine.Src != "" {
		if s.currentLine.Dst != "" {
			Connect_Analysis.ConnectAnalysis(s.currentLine.Src, s.currentLine.Dst, s.currentSystem)
		}
		for _, branchDst := range s.currentLine.Branches {
			Connect_Analysis.ConnectAnalysis(s.currentLine.Src, branchDst, s.currentSystem)
		}
	}	
}

// processPContent P 태그 종료 시 Name에 따라 처리:
//   - "Src"/"Dst": 라인 엔드포인트를 파싱하여 등록
//                  (Branch의 경우 Dst는 Branches에 추가).
//   - "TreatAsAtomicUnit": on → 현재 SubSystem을 원자로 간주 → Type=system 할당,
//                          그렇지 않으면 Type=class.
//   - "Port": 현재 블록에 포트 속성이 존재함을 표시.
//   - "Ports": 리스트(예: "[1,2,0]")를 파싱하여 portsFromList에 캐시
//              (통계 또는 이후 확장을 위해 사용).


func (s *parserState) processPContent() {
	switch s.currentPName {
	case "Src":
		s.currentLine.Src = extractID(s.currentPContent)
	case "Dst":
		dstID := extractID(s.currentPContent)
		if len(s.elementStack) >= 1 && s.elementStack[len(s.elementStack)-1].Name.Local == "Branch" {
			s.currentLine.Branches = append(s.currentLine.Branches, dstID)
		} else {
			s.currentLine.Dst = dstID
		}
	
	case "TreatAsAtomicUnit":
		s.currentBlock.IsAtomic = (s.currentPContent == "on")
	case "Port":
		s.hasPortAttr = true
	case "Ports":
		fields := strings.Trim(s.currentPContent, "[]")
		parts := strings.Split(fields, ",")
		s.portsFromList = nil
		for _, p := range parts {
			n, _ := strconv.Atoi(strings.TrimSpace(p))
			s.portsFromList = append(s.portsFromList, n)
		}
	}
}


// generateNewPath 현재 XML 경로와 SystemRef를 기반으로
// 하위 시스템 XML 경로를 생성한다.

func generateNewPath(originalPath, systemRef string) string {
	return filepath.Join(
		filepath.Dir(originalPath),
		systemRef+".xml",
	)
}

// extractID "id#…" 텍스트에서 id를 추출한다 ('#' 앞 부분).

func extractID(content string) string {
    parts := strings.Split(content, "#")
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}


// printSystemInfoToWriter 시스템 계층, 포트 통계, M1 값 및 컴포넌트 연결 등의 정보를
// 가독성 있는 형식으로 writer에 기록한다 (아이콘, 정렬 포함).
//
// 출력 요점:
//   - 시스템 행: 유형 아이콘(📦/🏷️), 이름 및 SID 표시;
//   - 통계 블록: nClass, portAsr(S-R과 C-S 가중), portSim(클래스 입출력 합),
//                M1 수식(classCount * adjustedPortCount * (inputs+outputs));
//   - 컴포넌트 연결: “소스 시스템/포트 → 대상 시스템/블록”을 항목별로 출력;
//   - 포트 및 연결: 각 포트의 연결 방향 [→]/[←] 표시;
//   - 하위 시스템 재귀 출력, 계층 들여쓰기와 분기 기호 유지.
//
// 주의: 이 함수는 builder에 텍스트를 포맷팅하는 역할만 하며,
//       파일을 직접 기록하지 않는다.

func printSystemInfoToWriter(system *M1_Public_Data.System, currentSystem *M1_Public_Data.System, indent string, writer *strings.Builder) {
	// Determine the output symbol based on the Type field
 	systemPrefix := "📦"


 	 	// Write system and statistical information
	 cleanName := strings.ReplaceAll(currentSystem.Name, "\n", " ")
	 cleanName = strings.ReplaceAll(cleanName, "\r", "")
	 
	 writer.WriteString(fmt.Sprintf("%sSystem: %s (SID: %s)\n", systemPrefix, cleanName, currentSystem.SID))

	//portCount := len(currentSystem.Port)
	inPortCount := 0
	outPortCount := 0
	cSCount := 0
	sRCount := 0
	classCount := 0
	classInputsSum := 0
	classOutputsSum := 0

	for _, port := range currentSystem.Port {
		if port.IO == "In" {
			inPortCount++
		} else if port.IO == "Out" {
			outPortCount++
		}
		if port.PortType == "C-S" {
			cSCount++
		} else {
			sRCount++
		}
	}

	adjustedPortCount := float64(sRCount) + float64(cSCount)*1.2

	for _, subSys := range currentSystem.System {
	classCount++
	classInputsSum += subSys.Inputs
	classOutputsSum += subSys.Outputs
}

    // Output statistical information
	writer.WriteString(fmt.Sprintf("%s  ├─📊 nSubsys: %d\n", indent, classCount))
writer.WriteString(fmt.Sprintf("%s  ├─📊 portAsr: %.1f (In: %d Out: %d, S-R: %d C-S: %d)\n",
	indent, adjustedPortCount, inPortCount, outPortCount, sRCount, cSCount))
writer.WriteString(fmt.Sprintf("%s  ├─📊 portSubsysIO: %d (In: %d Out: %d )\n",
	indent, classInputsSum+classOutputsSum, classInputsSum, classOutputsSum))
writer.WriteString(fmt.Sprintf("%s  ├─📊 M1: %.1f \n",
	indent, float64(classCount)*adjustedPortCount*float64(classInputsSum+classOutputsSum)))



	
	// Output the subsystem connections (component connections) initiated by the current system
	if len(currentSystem.ComponentConnections) > 0 {
		writer.WriteString(fmt.Sprintf("%s  ├─🧩 Subsystem connection:\n", indent))
		for _, conn := range currentSystem.ComponentConnections {
			srcName := findBlockNameBySID(system, conn.SrcPortSID)
			dstName := findBlockNameBySID(system, conn.DstBlockSID)
	
			// Determine the type of target component
			dstIcon := "📦"
	
			writer.WriteString(fmt.Sprintf("%s  │   └─📦 %s (SID: %s) → %s %s (SID: %s)\n",
				indent, srcName, conn.SrcPortSID, dstIcon, dstName, conn.DstBlockSID))
		}
	}
	

	// Output port information
	for _, port := range currentSystem.Port {
		writer.WriteString(fmt.Sprintf("%s  ├─🔌 Port: %-*s (SID: %s, Type: %-4s, IO: %-3s)\n",
			indent, portNameWidth, port.Name, port.SID, port.PortType, port.IO))

		// Output connection information
		if len(port.Connection) > 0 {
			var targets []string
			for _, conn := range port.Connection {
				var targetSID string
				var targetName string
				if conn.Direction == "out" {
					targetSID = conn.DstBlockSID
					targetName = findBlockNameBySID(system, conn.DstBlockSID)
					targets = append(targets, fmt.Sprintf("[→] %s (SID: %s)", targetName, targetSID))
				} else {
					targetSID = conn.SrcPortSID
					targetName = findBlockNameBySID(system, conn.SrcPortSID)
					targets = append(targets, fmt.Sprintf("[←] %s (SID: %s)", targetName, targetSID))
				}
			}
			if len(targets) > 1 {
				writer.WriteString(fmt.Sprintf("%s  │   └─🔗 [%d targets]  %s\n", indent, len(targets), strings.Join(targets, ", ")))
			} else {
				writer.WriteString(fmt.Sprintf("%s  │   └─🔗  %s\n", indent, targets[0]))
			}
		}
	}



	// Recursive output subsystem information
	for i, subSys := range currentSystem.System {
		last := (i == len(currentSystem.System)-1)
		prefix := "  └─"
		if !last {
			prefix = "  ├─"
		}
		writer.WriteString(fmt.Sprintf("%s%s", indent, prefix))
		printSystemInfoToWriter(system, subSys, indent+"    ", writer)
	}
}

// findBlockNameBySID 전체 시스템 트리에서 SID에 대응하는 이름을 탐색한다:
//   우선 포트와 매칭;
//   일반 Block과 매칭되면 "(Block) <Name>"을 반환하고 개행 정리를 수행;
//   시스템 자체일 경우 시스템명을 반환;
//   찾지 못하면 "Unknown (SID: <sid>)"을 반환한다.

func findBlockNameBySID(system *M1_Public_Data.System, sid string) string {
	for _, port := range system.Port {
		if port.SID == sid {
			return port.Name
		}
	}
	
	for _, block := range system.Block {
        if block.SID == sid {
            // Clean name by replacing newlines
            cleanName := strings.ReplaceAll(block.Name, "\n", " ")
            cleanName = strings.ReplaceAll(cleanName, "\r", "")
            return fmt.Sprintf("(Block) %s", strings.TrimSpace(cleanName))
        }
    }
	if system.SID == sid {
		return system.Name
	}
	for _, sub := range system.System {
		name := findBlockNameBySID(sub, sid)
		if name != "" && !strings.HasPrefix(name, "Unknown") {
			return name
		}
	}
	return fmt.Sprintf("Unknown (SID: %d)", sid)
}

// findBlockTypeBySID 시스템 트리에서 주어진 SID의 타입(class/system/...)을 탐색하여,
// 연결 표시 아이콘을 결정하는 데 사용한다.

func findBlockTypeBySID(system *M1_Public_Data.System, sid string) string {
	if system.SID == sid {
		return system.Type
	}
	for _, sub := range system.System {
		if sub.SID == sid {
			return sub.Type
		}
		if t := findBlockTypeBySID(sub, sid); t != "" {
			return t
		}
	}
	return ""
}

