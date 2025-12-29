package File_Utils_M2

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"FCU_Tools/Public_data"
)

// CheckAndSetM2InputPath는 지정된 디렉터리에 M2에 필요한 입력 파일이 포함되어 있는지 검사합니다.
// (complexity.json과 rq_versus_component.csv)이며, Public_data에 해당 경로를 저장합니다.
//
// 절차:
//   1) dir/complexity.json과 dir/rq_versus_component.csv를 조합합니다.
//   2) os.Stat을 호출해 파일 존재 여부를 확인하고, 누락 시 오류를 반환합니다.
//   3) 경로를 각각 Public_data.M2ComplexityJsonPath, Public_data.M2RqExcelPath에 저장합니다.
// func CheckAndSetM2InputPath(dir string) error {
// 	complexity := filepath.Join(dir, "complexity.json")
// 	rqCsv := filepath.Join(dir, "rq_versus_component.csv")

// 	if _, err := os.Stat(complexity); os.IsNotExist(err) {
// 		return fmt.Errorf("complexity.json을 찾을 수 없습니다: %s", complexity)
// 	}
// 	if _, err := os.Stat(rqCsv); os.IsNotExist(err) {
// 		return fmt.Errorf("rq_versus_component.csv을 찾을 수 없습니다: %s", rqCsv)
// 	}

// 	Public_data.M2ComplexityJsonPath = complexity
// 	// 변수명은 기존 그대로 사용하지만, 이제 CSV 경로를 담는다.
// 	Public_data.M2RqExcelPath = rqCsv
// 	return nil
// }

// PrepareM2OutputDir는 M2의 출력 디렉터리를 준비합니다.
//
// 절차:
//   1) 현재 작업 디렉터리를 가져옵니다.
//   2) <작업 디렉터리>/M2/output 경로를 조합합니다.
//   3) output/이 이미 존재하면 먼저 삭제한 뒤 새로 생성합니다.
//   4) 해당 경로를 Public_data.M2OutputlPath에 저장합니다.
func PrepareM2OutputDir() error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("작업 디렉토리를 가져오지 못했습니다: %v", err)
	}
	outputPath := filepath.Join(basePath, "M2", "output")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 output 디렉토리를 삭제하지 못했습니다: %v", err)
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("output 디렉터리를 만드는 데 실패했습니다: %v", err)
	}

	// Public_data에 경로 저장 변수
	Public_data.M2OutputlPath = outputPath

	return nil
}

// GenerateM2LDIXml complexity.json과 rq_versus_component.csv를 읽어 M2.ldi.xml을 생성한다.
//
// 프로세스:
//   1) complexity.json을 읽어 map[string]float64로 파싱 (모듈명 → 복잡도 값).
//   2) rq_versus_component.csv를 열고 모든 행을 읽어 Req 이름을 컴포넌트명에 매핑.
//   3) 정규식을 이용해 JSON key의 접두어([REQ] 형태)를 매칭하고,
//      excelMap을 활용해 컴포넌트명으로 매핑.
//
func GenerateM2LDIXml() error {
	// complexity.json 읽기
	data, err := ioutil.ReadFile(Public_data.M2ComplexityJsonPath)
	if err != nil {
		return fmt.Errorf("complexity.json 읽기 실패: %v", err)
	}

	var jsonMap map[string]float64
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return fmt.Errorf("complexity.json 살펴보기 실패: %v", err)
	}

	// CSV 파일 열기 (rq_versus_component.csv)
	f, err := os.Open(Public_data.M2RqExcelPath)
	if err != nil {
		return fmt.Errorf("CSV 열기 실패: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// 각 행의 컬럼 수가 달라도 읽을 수 있도록 설정
	r.FieldsPerRecord = -1

	excelRows, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("CSV 행 읽기 실패: %v", err)
	}

	excelMap := make(map[string]string)
	for _, row := range excelRows {
		if len(row) >= 2 {
			excelMap[strings.TrimSpace(row[0])] = row[1]
		}
	}

	type Property struct {
		XMLName xml.Name `xml:"property"`
		Name    string   `xml:"name,attr"`
		Value   string   `xml:",chardata"`
	}
	type Element struct {
		XMLName  xml.Name  `xml:"element"`
		Name     string    `xml:"name,attr"`
		Property []Property `xml:"property"`
	}
	type Root struct {
		XMLName xml.Name  `xml:"ldi"`
		Items   []Element `xml:"element"`
	}

	var result Root
	re := regexp.MustCompile(`^\[[^\]]+\]`)
	for key, val := range jsonMap {
		match := re.FindString(key)
		if compName, ok := excelMap[match]; ok {
			element := Element{
				Name: strings.ReplaceAll(compName, ".", ""),
				Property: []Property{{
					Name:  "coverage.m2",
					Value: fmt.Sprintf("%v", val),
				}},
			}
			result.Items = append(result.Items, element)
		}
	}

	outputFile := filepath.Join(Public_data.M2OutputlPath, "M2.ldi.xml")
	out, err := xml.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("XML 직렬화 실패: %v", err)
	}

	header := []byte(xml.Header)
	if err := ioutil.WriteFile(outputFile, append(header, out...), 0644); err != nil {
		return fmt.Errorf("ldi.xml 쓰기 실패: %v", err)
	}
	fmt.Printf("📄 M2 지표 계산 완료: %s\n", outputFile)
	return nil
}
