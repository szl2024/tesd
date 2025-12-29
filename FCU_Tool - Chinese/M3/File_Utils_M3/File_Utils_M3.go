package File_Utils_M3

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

// CheckAndSetM3InputPath는 M3에 필요한 입력 파일 경로를 확인하고 설정한다.
//
// 프로세스:
//   1) 사용자가 지정한 디렉터리에서 component_info.csv 파일을 찾는다.
//   2) 존재하면 경로를 Public_data.M3component_infoxlsxPath에 저장한다. (변수명은 호환성을 위해 유지)
//   3) 존재하지 않으면 오류를 반환하고 누락을 알린다.
func CheckAndSetM3InputPath(dir string) error {
	complexity := filepath.Join(dir, "component_info.csv")

	if _, err := os.Stat(complexity); os.IsNotExist(err) {
		return fmt.Errorf("component_info.csv를 찾을 수 없습니다: %s", complexity)
	}

	// 변수명은 기존 그대로지만, 이제 CSV 경로를 저장한다.
	Public_data.M3component_infoxlsxPath = complexity
	return nil
}

// PrepareM2OutputDir는 M3의 출력 디렉터리를 준비한다.
//
// 프로세스:
//   1) 현재 작업 디렉터리를 가져온다.
//   2) 경로를 <작업 디렉터리>/M3/output으로 결합한다.
//   3) output이 이미 존재하면 삭제 후 새로 생성한다.
//   4) 경로를 Public_data.M3OutputlPath에 저장한다.
func PrepareM2OutputDir() error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("작업 디렉토리를 가져오지 못했습니다.: %v", err)
	}
	outputPath := filepath.Join(basePath, "M3", "output")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 output 디렉토리를 삭제하지 못했습니다.: %v", err)
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("output 디렉터리를 만드는 데 실패했습니다.: %v", err)
	}

	// Public_data에 경로 저장 변수
	Public_data.M3OutputlPath = outputPath

	return nil
}

// GenerateM3LDIXml ASW 의존성과 component_info.csv를 읽어
// M3.ldi.xml 및 M3.txt를 생성한다.
//
// 프로세스:
//   1) SWC_Dependence.ExtractDependenciesRawFromASW 호출 → ASW 원시 의존성(컴포넌트 → 컴포넌트) 읽기.
//   2) component_info.csv 열기 → 각 컴포넌트의 Layer 값을 읽어 layerMap에 저장.
//   3) 의존성 순회:
//        - 각 컴포넌트의 소스 의존 개수(sourceCount) 집계.
//        - 규칙 위반 시 (fromLayer > toLayer 이고 인터페이스=CS, 또는 레벨 차이 > 1) → violation으로 기록,
//          M3.txt에 "from-->to" 한 줄 작성.
//   4) 각 컴포넌트에 대해 <element name="..."> 생성, 포함 항목:
//        - coverage.m3 = 위반 횟수
//        - coverage.m3demo = 전체 의존 횟수
//   5) LDI 파일을 M3/output/M3.ldi.xml에 출력하고 완료 메시지 출력.
func GenerateM3LDIXml() error {
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

	dependencies, err := SWC_Dependence.ExtractDependenciesRawFromASW(Public_data.ConnectorFilePath)
	if err != nil {
		return fmt.Errorf("ASW 종속성 읽기 실패: %v", err)
	}

	// component_info.csv 읽기
	f, err := os.Open(Public_data.M3component_infoxlsxPath)
	if err != nil {
		return fmt.Errorf("component_info.csv 열기 실패: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// 각 행의 컬럼 수가 달라도 읽을 수 있도록 설정
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("component_info.csv 읽기 실패: %v", err)
	}

	layerMap := make(map[string]int)
	// 첫 행은 헤더라고 가정하고 rows[1:]부터 처리 (기존 xlsx 로직과 동일)
	for _, row := range rows[1:] {
		if len(row) >= 4 {
			name := strings.TrimSpace(row[0])
			layerStr := strings.TrimSpace(row[3])
			var layer int
			fmt.Sscanf(layerStr, "%d", &layer)
			layerMap[name] = layer
		}
	}

	m3TxtPath := filepath.Join(Public_data.M3OutputlPath, "M3.txt")
	if err := os.Remove(m3TxtPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("기존 M3.txt 삭제 실패: %v", err)
	}

	violationMap := make(map[string]int)
	sourceCount := make(map[string]int)

	for from, deps := range dependencies {
		fromLayer, fromOk := layerMap[from]
		for _, dep := range deps {
			to := dep.To
			count := dep.Count
			//ifType := dep.InterfaceType

			toLayer, toOk := layerMap[to]
			if !fromOk || !toOk {
				continue
			}

			sourceCount[from] += count

			absDiff := fromLayer - toLayer
			if absDiff < 0 {
				absDiff = -absDiff
			}
			// 디버그 출력은 주석 처리
			// fmt.Printf("🔍 CHECK: %s (ASIL %d) → %s (ASIL %d), IF: %s, DIFF: %d\n", from, fromLayer, to, toLayer, ifType, absDiff)

			if (fromLayer > toLayer) || (absDiff > 1) {
				// fmt.Println("🚨 VIOLATION")
				violationMap[from] += count
				line := fmt.Sprintf("%s-->%s\n", from, to)
				f, err := os.OpenFile(m3TxtPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("M3.txt 파일 열기 실패: %v", err)
				}
				if _, err := f.WriteString(line); err != nil {
					f.Close()
					return fmt.Errorf("M3.txt 기록 실패: %v", err)
				}
				f.Close()
			} else {
				// fmt.Println("✅ OK: No violation")
			}
		}
	}

	var result Root
	for comp, demoCount := range sourceCount {
		violationCount := violationMap[comp]
		elem := Element{
			Name: comp,
			Property: []Property{
				{Name: "coverage.m3", Value: fmt.Sprintf("%d", violationCount)},
				{Name: "coverage.m3demo", Value: fmt.Sprintf("%d", demoCount)},
			},
		}
		result.Items = append(result.Items, elem)
	}

	outPath := filepath.Join(Public_data.M3OutputlPath, "M3.ldi.xml")
	output, err := xml.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("XML 생성 실패: %v", err)
	}
	header := []byte(xml.Header)
	if err := ioutil.WriteFile(outPath, append(header, output...), 0644); err != nil {
		return fmt.Errorf("M3.ldi.xml 저장 실패: %v", err)
	}

	fmt.Println("📄 M3 및 m3demo 지표 계산 완료:", outPath)
	return nil
}
