package System_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/Port_Analysis"
)

// SubSystem의 Name/SID/Level/BlockType을 저장하는 데 사용됩니다.
type SubSystemInfo struct {
	Name      string
	SID       string
	Level     int
	BlockType string
}

// P 태그
type xmlP struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

// PortCounts 태그
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

// ======================== 외부 입력 포트 ================================
// fatherName: 현재 system_xxx.xml에 해당하는 부모 노드 이름(L1은 빈 문자열)
func AnalyzeSubSystemsInFile(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	switch level {
	case 1:
		return analyzeSubSystemsLevel1(dir, file, level, fatherName)
	case 2:
		return analyzeSubSystemsLevel2(dir, file, level, fatherName)
	case 3:
		return analyzeSubSystemsLevel3(dir, file, level, fatherName)
	default:
		// 3층 및 이후는 모두 “Inport/Outport가 아닌 Block”으로 통일하여 처리합니다.
		return analyzeSubSystemsLevel3(dir, file, level, fatherName)
	}
}
//여기서는 원래 L1과 L2 레이어를 두 개의 함수로 각각 분석해야 하지만, 분석 함수 안에서 이미 구분 로직이 있으므로 L1과 L2는 동일한 함수를 사용합니다.
// ======================== 로직 1(L1: 유효하지 않은 SubSystem 필터링) ================================
func analyzeSubSystemsLevel1(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, true, fatherName)
}

// ======================== 로직 2(L2: SubSystem을 필터링하지 않음) ================================
func analyzeSubSystemsLevel2(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeSubSystemsCommon(dir, file, level, false, fatherName)
}

// ======================== 로직 3(L3+: Inport/Outport가 아닌 Block) =========================
func analyzeSubSystemsLevel3(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	return analyzeNonPortBlocks(dir, file, level, fatherName)
}

// ======================== 범용 SubSystem 분석(재귀 제거, 외부에서 제어) ====================
func analyzeSubSystemsCommon(dir, file string, level int, applyLevel1Filter bool, fatherName string) ([]SubSystemInfo, error) {
	
	//분석할 파일의 경로를 조합합니다.
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("XML 읽기 실패 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("XML 파싱 실패 [%s]: %w", fullPath, err)
	}

	// 모델명을 추출합니다: BuildDir/<Model>/simulink/systems → <Model>
	modelDir := filepath.Dir(filepath.Dir(dir))
	modelName := filepath.Base(modelDir)

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {

		if b.BlockType != "SubSystem" {
			continue
		}

		// === level=1일 때 필터링: Ports가 비어 있거나 PortCounts가 비어 있는 SubSystem은 바로 건너뜁니다 ===
		if applyLevel1Filter && level == 1 {
			invalid := false
			// (1)과 (2) 중 하나라도 충족하지 못하면 부적격 블록이며, 즉 1층에서는 초기화된 SubSystem이 존재하므로 필터링해야 합니다.
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

			// (2) PortCounts 태그가 존재하지만 비어 있습니다.
			if !invalid && b.PortCounts != nil {
				if b.PortCounts.In == "" && b.PortCounts.Out == "" && b.PortCounts.Trigger == "" {
					invalid = true
				}
			}

			if invalid {
				continue
			}
		}

		// 이름을 한 번 정리하여 줄바꿈과 여러 공백을 제거합니다.
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

	// 이 레이어에서 출력할 BlockSID 목록을 Port_Analysis에 전달하여, 그쪽에서 Block → Port 순서로 통일해 출력하도록 합니다.
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 분석 실패 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}

// ======================== Inport/Outport가 아닌 Block 분석(3층 및 이후) ==================
// 지정된 system_xxx.xml에서 BlockType이 "Inport"가 아니고 "Outport"도 아닌 모든 Block을 찾습니다.
// 이들 Block의 Name/BlockType/SID를 기록하고, Port_Analysis에 전달해 통일 출력합니다.
func analyzeNonPortBlocks(dir, file string, level int, fatherName string) ([]SubSystemInfo, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("XML 읽기 실패 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("XML 파싱 실패 [%s]: %w", fullPath, err)
	}

	// 모델명을 추출합니다: BuildDir/<Model>/simulink/systems → <Model>
	modelDir := filepath.Dir(filepath.Dir(dir))
	modelName := filepath.Base(modelDir)

	var result []SubSystemInfo
	var blockSIDs []string

	for _, b := range sys.Blocks {

		// Inport와 Outport는 건너뜁니다.
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

	// Port_Analysis에 넘겨 Block + Port를 통일된 형식으로 출력합니다.
	if len(blockSIDs) > 0 && modelName != "" {
		if err := Port_Analysis.AnalyzePortsInFile(dir, file, level, modelName, fatherName, blockSIDs); err != nil {
			fmt.Printf("⚠️ Port_Analysis 분석 실패 [%s]: %v\n", fullPath, err)
		}
	}

	return result, nil
}
