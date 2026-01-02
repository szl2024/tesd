package File_Utils_M4

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/SWC_Dependence"
	"FCU_Tools/Public_data"
)

// PrepareM4OutputDir M4의 출력 디렉터리를 초기화하고 준비한다.
//
// 프로세스:
//   1) 현재 작업 디렉터리 basePath를 가져온다.  
//   2) <basePath>/M4/output 경로를 생성한다.  
//   3) output 디렉터리가 이미 존재하면 삭제 후 새로 만든다.  
//   4) 경로를 Public_data.M4OutputlPath에 저장하여 이후 모듈에서 사용한다.  
func PrepareM4OutputDir() error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("작업 디렉토리를 가져오지 못했습니다: %v", err)
	}
	outputPath := filepath.Join(basePath, "M4", "output")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 output 디렉토리를 삭제하지 못했습니다.: %v", err)
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("output 디렉터리를 만드는 데 실패했습니다: %v", err)
	}

	// Public_data에 경로 저장 변수
	Public_data.M4OutputlPath = outputPath

	return nil
}

// GenerateM4LDIXml ASW 연결 의존성과 component_info.csv을 읽어
// M4 지표를 계산하고 M4.ldi.xml 및 M4.txt를 생성한다.
//
// 계산 로직:
//   1) SWC_Dependence.ExtractDependenciesRawFromASW 호출 → 모든 컴포넌트 연결을 읽는다 (원시 연결 정보 유지).  
//   2) component_info.csv 열기 → 컴포넌트의 Manager 및 Layer 정보를 읽어 compMap에 저장한다.  
//   3) 의존성 순회:  
//        - 각 컴포넌트의 sourceCount(의존 총수)를 갱신한다.  
//        - 위반 여부 검사:  
//            * 같은 Layer인데 Manager가 다르면 → 위반.  
//            * Cross Layer인 경우:  
//                - from Layer > to Layer이고 from.Manager != to → 위반.  
//                - from Layer < to Layer이고 to.Manager != from → 위반.  
//        - 위반 발생 시: violationMap[from]에 횟수를 누적하고, M4.txt에 "from-->to" 한 줄 기록.  
//   4) 각 컴포넌트에 대해 LDI 요소를 생성, 두 가지 속성 포함:  
//        - coverage.m4     = 위반 연결 수  
//        - coverage.m4demo = 전체 의존 수  
//   5) XML로 직렬화하여 M4/output/M4.ldi.xml에 출력한다.  
func GenerateM4LDIXml() error {
	type Property struct {
		XMLName xml.Name `xml:"property"`
		Name    string   `xml:"name,attr"`
		Value   string   `xml:",chardata"`
	}
	type Element struct {
		XMLName  xml.Name   `xml:"element"`
		Name     string     `xml:"name,attr"`
		Property []Property `xml:"property"`
	}
	type Root struct {
		XMLName xml.Name  `xml:"ldi"`
		Items   []Element `xml:"element"`
	}

	// 연결 정보를 로드합니다 (원본 연결 유지)
	connectorDeps, err := SWC_Dependence.ExtractDependenciesRawFromASW(Public_data.ConnectorFilePath)
	if err != nil {
		return fmt.Errorf("asw 종속성 읽기 실패: %v", err)
	}
	totalLinks := 0
	for _, deps := range connectorDeps {
		totalLinks += len(deps)
	}
	//fmt.Printf("🔗 총 연결 개수 로드됨: %d\n", totalLinks)

	// 컴포넌트 정보를 로드합니다 (component_info.csv)
	// 주의: Public_data.M3component_infoxlsxPath 변수명은 그대로지만, 실제로는 CSV 경로를 담고 있다.
	compFile, err := os.Open(Public_data.M3component_infoxlsxPath)
	if err != nil {
		return fmt.Errorf("component_info.csv 열기 실패: %v", err)
	}
	defer compFile.Close()

	csvReader := csv.NewReader(compFile)
	// 각 행의 컬럼 수가 달라도 읽을 수 있도록 설정
	csvReader.FieldsPerRecord = -1

	compRows, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("component_info.csv 컨텐츠를 읽지 못했습니다: %v", err)
	}

	type CompMeta struct {
		Manager string
		Layer   int
	}
	compMap := make(map[string]CompMeta)
	// 첫 행은 헤더라고 가정하고 compRows[1:]부터 처리
	for _, row := range compRows[1:] {
		if len(row) >= 4 {
			name := strings.TrimSpace(row[0])
			manager := strings.TrimSpace(row[1])
			var layer int
			fmt.Sscanf(strings.TrimSpace(row[3]), "%d", &layer)
			compMap[name] = CompMeta{Manager: manager, Layer: layer}
		}
	}

	m4TxtPath := filepath.Join(Public_data.M4OutputlPath, "M4.txt")
	if err := os.Remove(m4TxtPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("기존 M4.txt 삭제 실패: %v", err)
	}

	violationMap := make(map[string]int)
	sourceCount := make(map[string]int)

	for from, deps := range connectorDeps {
		fromMeta, fromOk := compMap[from]
		for _, dep := range deps {
			to := dep.To
			count := dep.Count
			toMeta, toOk := compMap[to]

			//fmt.Printf("🔍 CHECK: %s (%d, M:%s) → %s (%d, M:%s)\n",from, fromMeta.Layer, fromMeta.Manager,to, toMeta.Layer, toMeta.Manager)

			if !fromOk || !toOk {
				fmt.Println("⚠️ 컴포넌트 메타 정보 누락. 스킵합니다.")
				continue
			}

			sourceCount[from] += count
			violation := false

			if fromMeta.Layer == toMeta.Layer {
				if fromMeta.Manager != toMeta.Manager {
					violation = true
				}
			} else {
				if fromMeta.Layer > toMeta.Layer {
					if fromMeta.Manager != to {
						violation = true
					}
				} else {
					if toMeta.Manager != from {
						violation = true
					}
				}
			}

			if violation {
				//fmt.Printf("🚨 Violation 발생: %s → %s\n", from, to)
				violationMap[from] += count
				line := fmt.Sprintf("%s-->%s\n", from, to)
				f, err := os.OpenFile(m4TxtPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("M4.txt 파일을 열 수 없습니다: %v", err)
				}
				if _, err := f.WriteString(line); err != nil {
					f.Close()
					return fmt.Errorf("M4.txt에 기록할 수 없습니다: %v", err)
				}
				f.Close()
			} else {
				//fmt.Printf("✅ OK: No violation\n")
			}
		}
	}

	var result Root
	for comp, demoCount := range sourceCount {
		violationCount := violationMap[comp]
		elem := Element{
			Name: comp,
			Property: []Property{
				{Name: "coverage.m4", Value: fmt.Sprintf("%d", violationCount)},
				{Name: "coverage.m4demo", Value: fmt.Sprintf("%d", demoCount)},
			},
		}
		result.Items = append(result.Items, elem)
	}

	outPath := filepath.Join(Public_data.M4OutputlPath, "M4.ldi.xml")
	output, err := xml.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("XML 컨텐트 생성 실패: %v", err)
	}
	header := []byte(xml.Header)
	if err := ioutil.WriteFile(outPath, append(header, output...), 0644); err != nil {
		return fmt.Errorf("M4.ldi.xml 파일을 쓰는 데 실패했습니다: %v", err)
	}
	fmt.Println("📄 M4 및 m4demo 지표 계산 완료:", outPath)
	return nil
}
