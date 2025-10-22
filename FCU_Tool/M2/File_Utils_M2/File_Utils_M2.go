package File_Utils_M2

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"github.com/xuri/excelize/v2"
	"FCU_Tools/Public_data"
)

// CheckAndSetM2InputPath 检查指定目录下是否包含 M2 所需的输入文件
// （complexity.json 与 rq_versus_component.xlsx），并在 Public_data 中保存其路径。
// 
// 流程：
//   1) 拼接 dir/complexity.json 与 dir/rq_versus_component.xlsx。
//   2) 调用 os.Stat 确认文件存在；缺失则返回错误。
//   3) 将路径分别存入 Public_data.M2ComplexityJsonPath、Public_data.M2RqExcelPath。
func CheckAndSetM2InputPath(dir string) error {
	complexity := filepath.Join(dir, "complexity.json")
	rqExcel := filepath.Join(dir, "rq_versus_component.xlsx")

	if _, err := os.Stat(complexity); os.IsNotExist(err) {
		return fmt.Errorf("complexity.json을 찾을 수 없습니다: %s", complexity)
	}
	if _, err := os.Stat(rqExcel); os.IsNotExist(err) {
		return fmt.Errorf("rq_versus_component.xlsx을 찾을 수 없습니다: %s", rqExcel)
	}

	Public_data.M2ComplexityJsonPath = complexity
	Public_data.M2RqExcelPath = rqExcel
	return nil
}

// PrepareM2OutputDir 准备 M2 的输出目录。
// 
// 流程：
//   1) 获取当前工作目录。
//   2) 拼接 <工作目录>/M2/output 路径。
//   3) 若已存在 output/ 则先删除再新建。
//   4) 将路径保存到 Public_data.M2OutputlPath。
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

// GenerateM2LDIXml complexity.json과 rq_versus_component.xlsx를 읽어 M2.ldi.xml을 생성한다.
//
// 프로세스:
//   1) complexity.json을 읽어 map[string]float64로 파싱 (모듈명 → 복잡도 값).
//   2) rq_versus_component.xlsx를 열고 Sheet1을 읽어 Req 이름을 컴포넌트명에 매핑.
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

	// Excel 파일 열기
	excelFile, err := excelize.OpenFile(Public_data.M2RqExcelPath)
	if err != nil {
		return fmt.Errorf("Excel 열기 실패: %v", err)
	}

	excelMap := make(map[string]string)
	excelRows, err := excelFile.GetRows("Sheet1")
	if err != nil {
		return fmt.Errorf("Excel 행 읽기 실패: %v", err)
	}
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

	return nil
}