package File_Utils_M5

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/Public_data"
)

// PrepareM5OutputDir M5의 출력 디렉터리를 초기화하고 준비한다.
//
// 프로세스:
//   1) 현재 작업 디렉터리 basePath를 가져온다.  
//   2) <basePath>/M5/output 경로를 생성한다.  
//   3) output 디렉터리가 이미 존재하면 삭제 후 새로 만든다.  
//   4) 경로를 Public_data.M5OutputlPath에 저장한다.  
func PrepareM5OutputDir() error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("작업 디렉토리를 가져오지 못했습니다: %v", err)
	}
	outputPath := filepath.Join(basePath, "M5", "output")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 output 디렉토리를 삭제하지 못했습니다: %v", err)
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("output 디렉터리를 만드는 데 실패했습니다: %v", err)
	}

	// Public_data에 경로 저장 변수
	Public_data.M5OutputlPath = outputPath

	return nil
}

// GenerateM5LDIXml component_info.csv을 읽어
// M5.ldi.xml을 생성한다 (m5 및 m5demo 속성 포함).
//
// 계산 로직:
//   1) component_info.csv을 열고 내용을 읽는다.  
//   2) 두 번째 행부터 읽는다:  
//        - row[0] = 컴포넌트 이름  
//        - row[4] = ASIL 분리 여부(Y/N)  
//   3) asilSplit == "Y"이면 coverage.m5 = 1, 그렇지 않으면 = 0.  
//   4) 각 컴포넌트에 대해 coverage.m5demo = 1을 고정 추가한다 (데모용 기준값).  
//   5) 모든 컴포넌트를 <element name="..."><property .../></element> 형태로 변환하여  
//      M5/output/M5.ldi.xml에 기록한다.  
func GenerateM5LDIXml() error {
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

	// component_info.csv 열기
	// 주의: Public_data.M3component_infoxlsxPath 변수명은 그대로지만,
	// 실제로는 component_info.csv 경로를 담고 있다(M3/M4와 동일 패턴).
	compInfoFile, err := os.Open(Public_data.M3component_infoxlsxPath)
	if err != nil {
		return fmt.Errorf("component_info.csv 열기 실패: %v", err)
	}
	defer compInfoFile.Close()

	reader := csv.NewReader(compInfoFile)
	// 각 행의 컬럼 수가 달라도 읽을 수 있도록 설정
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("component_info.csv 컨텐츠를 읽지 못했습니다.: %v", err)
	}

	var result Root
	// 첫 행은 헤더라고 가정하고 rows[1:]부터 처리 (기존 xlsx 로직과 동일)
	for _, row := range rows[1:] {
		if len(row) >= 5 {
			name := strings.TrimSpace(row[0])
			asilSplit := strings.TrimSpace(row[4])

			m5 := "0"
			if asilSplit == "Y" {
				m5 = "1"
			}

			elem := Element{
				Name: name,
				Property: []Property{
					{
						Name:  "coverage.m5",
						Value: m5,
					},
					{
						Name:  "coverage.m5demo",
						Value: "1",
					},
				},
			}
			result.Items = append(result.Items, elem)
		}
	}

	// XML 파일 쓰기
	outPath := filepath.Join(Public_data.M5OutputlPath, "M5.ldi.xml")
	output, err := xml.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("XML 컨텐트 생성 실패: %v", err)
	}
	header := []byte(xml.Header)
	if err := ioutil.WriteFile(outPath, append(header, output...), 0644); err != nil {
		return fmt.Errorf("M5 ldi.xml 파일 쓰기 실패: %v", err)
	}

	fmt.Println("📄 M5 및 m5demo 지표 계산 완료:", outPath)
	return nil
}
