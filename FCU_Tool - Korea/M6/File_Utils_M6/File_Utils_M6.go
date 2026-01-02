package File_Utils_M6

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

// PrepareM6OutputDir M6의 출력 디렉터리를 초기화하고 준비한다.
//
// 프로세스:
//   1) 현재 작업 디렉터리 basePath를 가져온다.  
//   2) <basePath>/M6/output 경로를 생성한다.  
//   3) output 디렉터리가 이미 존재하면 삭제 후 새로 만든다.  
//   4) 경로를 Public_data.M6OutputlPath에 저장한다.  
func PrepareM6OutputDir() error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("작업 디렉토리를 가져오지 못했습니다: %v", err)
	}
	outputPath := filepath.Join(basePath, "M6", "output")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 output 디렉토리를 삭제하지 못했습니다.: %v", err)
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("output 디렉터리를 만드는 데 실패했습니다.: %v", err)
	}

	// Public_data에 경로 저장 변수
	Public_data.M6OutputlPath = outputPath

	return nil
}

// GenerateM6LDIXml component_info.csv의 ASIL 등급과 ASW 의존 관계를 읽어
// M6 지표를 계산하고 M6.ldi.xml 및 M6.txt를 생성한다.
//
// 계산 로직:
//   1) component_info.csv을 열고 3번째 열(ASIL 등급 A/B/C/D)을 읽어
//      숫자 등급 1~4로 매핑하여 asilLevelMap에 저장한다.  
//   2) SWC_Dependence.ExtractDependenciesRawFromASW 호출 → 컴포넌트 의존성(from→to, 연결 횟수와 인터페이스 타입 포함) 읽기.  
//   3) 의존성 순회:  
//        - 각 from 컴포넌트의 총 의존 수(sourceCount)를 집계한다.  
//        - 만약 from의 ASIL 등급 < to의 ASIL 등급이면 → 위반으로 판정:  
//            * violationMap[from] += count  
//            * M6.txt에 "from (ASIL x) → to (ASIL y)" 한 줄 기록  
//   4) 통계 결과를 기반으로 각 컴포넌트에 대해 LDI 요소 생성, 다음 속성 포함:  
//        - coverage.m6     = 위반 의존 횟수  
//        - coverage.m6demo = 전체 의존 횟수  
//   5) 결과를 M6/output/M6.ldi.xml에 출력한다.  
func GenerateM6LDIXml() error {
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

	//  Step 1: component_info.csv에서 ASIL 등급(3열) 추출
	//  (변수명은 *.xlsx지만, 실제로는 component_info.csv 경로가 들어 있음: M3/M4/M5와 동일 패턴)
	asilFile, err := os.Open(Public_data.M3component_infoxlsxPath)
	if err != nil {
		return fmt.Errorf("component_info.csv 열기 실패: %v", err)
	}
	defer asilFile.Close()

	reader := csv.NewReader(asilFile)
	// 각 행의 컬럼 수가 달라도 읽을 수 있도록 설정
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("component_info.csv 컨텐츠를 읽지 못했습니다: %v", err)
	}

	asilMap := map[string]int{"A": 1, "B": 2, "C": 3, "D": 4}
	asilLevelMap := make(map[string]int)

	// 첫 행은 헤더라고 가정하고 rows[1:]부터 처리 (기존 xlsx 로직과 동일)
	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		component := strings.TrimSpace(row[0])
		asil := strings.ToUpper(strings.TrimSpace(row[2]))
		if level, ok := asilMap[asil]; ok {
			asilLevelMap[component] = level
		}
	}

	//  Step 2: 의존성 읽기(각 연결마다)
	connectorDeps, err := SWC_Dependence.ExtractDependenciesRawFromASW(Public_data.ConnectorFilePath)
	if err != nil {
		return fmt.Errorf("asw 연결 분석 실패: %v", err)
	}

	violationMap := make(map[string]int)
	sourceCount := make(map[string]int)

	//  M6.txt 파일은 위반 연결을 기록합니다.
	m6TxtPath := filepath.Join(Public_data.M6OutputlPath, "M6.txt")
	_ = os.Remove(m6TxtPath)

	for from, targets := range connectorDeps {
		fromLevel, fromOk := asilLevelMap[from]
		for _, dep := range targets {
			to := dep.To
			count := dep.Count
			toLevel, toOk := asilLevelMap[to]

			sourceCount[from] += count
			// 디버그용 출력은 주석 처리
			// fmt.Printf("🔍 CHECK: %s (ASIL %d) → %s (ASIL %d), Count: %d\n", from, fromLevel, to, toLevel, count)

			if fromOk && toOk {
				if fromLevel < toLevel {
					// fmt.Printf("🚨 VIOLATION DETECTED: %s → %s\n", from, to)
					violationMap[from] += count

					line := fmt.Sprintf("%s (ASIL %d) → %s (ASIL %d)\n", from, fromLevel, to, toLevel)
					f, err := os.OpenFile(m6TxtPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err == nil {
						_, _ = f.WriteString(line)
						f.Close()
					}
				} else {
					// fmt.Printf("✅ OK: No violation\n")
				}
			} else {
				fmt.Printf("⚠️ ASIL level not found for %s or %s\n", from, to)
			}
		}
	}

	//  Step 3: XML 출력 생성
	var result Root
	for name, count := range sourceCount {
		violation := violationMap[name]
		elem := Element{
			Name: name,
			Property: []Property{
				{Name: "coverage.m6", Value: fmt.Sprintf("%d", violation)},
				{Name: "coverage.m6demo", Value: fmt.Sprintf("%d", count)},
			},
		}
		result.Items = append(result.Items, elem)
	}

	outPath := filepath.Join(Public_data.M6OutputlPath, "M6.ldi.xml")
	output, err := xml.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("XML 직렬화 실패: %v", err)
	}
	header := []byte(xml.Header)
	if err := ioutil.WriteFile(outPath, append(header, output...), 0644); err != nil {
		return fmt.Errorf("M6.ldi.xml 쓰기 실패: %v", err)
	}

	fmt.Println("📄 M6 및 m6demo 지표 계산 완료:", outPath)
	return nil
}
