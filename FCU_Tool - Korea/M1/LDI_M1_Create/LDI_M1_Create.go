package LDI_M1_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/Public_data"
)

// XML 구조 정의
type Property struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",chardata"`
}

type Uses struct {
	XMLName  xml.Name `xml:"uses"`
	Provider string   `xml:"provider,attr"`
	Strength string   `xml:"strength,attr,omitempty"`
}

type Element struct {
	XMLName  xml.Name   `xml:"element"`
	Name     string     `xml:"name,attr"`
	Uses     []Uses     `xml:"uses"`
	Property []Property `xml:"property"`
}

type Root struct {
	XMLName xml.Name  `xml:"ldi"`
	Items   []Element `xml:"element"`
}

// MergeM1ToMainLDI
// LDIDir 디렉터리에서 M1 단계에 생성된 모든 *.ldi.xml의 coverage.m1을
// 주 LDI(Output/result.ldi.xml)에 병합합니다.
//
// 설명:
// - 가정: M1의 *.ldi.xml은 생성 단계에서 이미 “모델명 변경”(예: GenerateM1LDIFromTxt가 txt 파일명을 모델명으로 사용)을 완료했다.
// - 따라서 여기서는 더 이상 asw.csv를 읽지 않고, runnable→모델명 매핑도 수행하지 않으며, M1의 ldi.xml을 제자리에서 수정하지도 않는다.
func MergeM1ToMainLDI() error {
	// 1) 주 LDI 경로 확정
	if Public_data.OutputDir == "" {
		return fmt.Errorf("주 LDI 출력 디렉터리가 초기화되지 않았습니다. 먼저 InitOutputDirectory를 호출하세요.")
	}
	mainLDIPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")

	// 2) 주 LDI 읽기
	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패 [%s]: %v", mainLDIPath, err)
	}

	var mainRoot Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI XML 파싱 실패: %v", err)
	}

	// 3) M1 LDI 디렉터리를 스캔하여 coverage.m1을 수집합니다(키는 el.Name을 그대로 사용).
	m1Dir := M1_Public_Data.LDIDir
	if m1Dir == "" {
		return fmt.Errorf("M1_Public_Data.LDIDir가 설정되지 않아 M1의 LDI 파일 디렉터리를 찾을 수 없습니다.")
	}

	entries, err := os.ReadDir(m1Dir)
	if err != nil {
		return fmt.Errorf("M1 LDI 디렉터리 읽기 실패 [%s]: %v", m1Dir, err)
	}

	m1Map := make(map[string]string) // element name → coverage.m1

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".ldi.xml") {
			continue
		}

		path := filepath.Join(m1Dir, e.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("⚠️ M1 LDI 파일 읽기 실패 [%s]: %v\n", path, err)
			continue
		}

		var m1Root Root
		if err := xml.Unmarshal(data, &m1Root); err != nil {
			fmt.Printf("⚠️ M1 LDI 파일 파싱 실패 [%s]: %v\n", path, err)
			continue
		}

		for _, el := range m1Root.Items {
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					m1Map[el.Name] = p.Value
				}
			}
		}
	}

	if len(m1Map) == 0 {
		fmt.Println("ℹ️ M1 LDI 디렉터리에서 coverage.m1 속성을 하나도 찾지 못해 주 LDI를 수정하지 않습니다.")
		return nil
	}

	// 4) 주 LDI의 인덱스 테이블을 구축합니다: element name → index
	mainIndex := make(map[string]int)
	for i, el := range mainRoot.Items {
		mainIndex[el.Name] = i
	}

	// 5) coverage.m1을 주 LDI에 병합합니다.
	for name, val := range m1Map {
		if idx, ok := mainIndex[name]; ok {
			// 주 LDI에 해당 element가 이미 존재합니다: coverage.m1이 이미 있는지 확인합니다.
			el := &mainRoot.Items[idx]
			exists := false
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			el.Property = append(el.Property, Property{
				Name:  "coverage.m1",
				Value: val,
			})
		} else {
			// 주 LDI에 해당 element가 없습니다: coverage.m1만 포함한 element를 새로 추가합니다.
			newEl := Element{
				Name: name,
				Property: []Property{
					{
						Name:  "coverage.m1",
						Value: val,
					},
				},
			}
			mainRoot.Items = append(mainRoot.Items, newEl)
			mainIndex[name] = len(mainRoot.Items) - 1
		}
	}

	// 6) 주 LDI에 다시 씁니다.
	out, err := xml.MarshalIndent(mainRoot, "  ", "    ")
	if err != nil {
		return fmt.Errorf("주 LDI XML 직렬화 실패: %v", err)
	}

	header := []byte(xml.Header)
	if err := ioutil.WriteFile(mainLDIPath, append(header, out...), 0644); err != nil {
		return fmt.Errorf("주 LDI 파일 쓰기 실패 [%s]: %v", mainLDIPath, err)
	}
	
	fmt.Println("✅ M1지표 병합 성공")
	return nil
}
