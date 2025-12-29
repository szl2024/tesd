package LDI_M5_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"FCU_Tools/Public_data"
)

// MergeM5ToMainLDI M5.ldi.xml의 m5 및 m5demo 지표를
// 주 LDI 파일 result.ldi.xml에 병합한다.
//
// 프로세스:
//   1) 주 LDI 파일(OutputDir/result.ldi.xml)과 M5/output/M5.ldi.xml을 읽는다.  
//   2) XML을 파싱하여 m5Map[name] → []Property를 구성한다.  
//   3) 주 LDI 요소를 순회하면서: 컴포넌트가 m5Map에 있으면 기존 속성을 확인하고, 누락된 속성은 추가한다.  
//   4) XML을 다시 직렬화하여 주 LDI 파일에 덮어쓴다.  
 func MergeM5ToMainLDI() error {
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

	mainLDIPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")
	m5LDIPath := filepath.Join(Public_data.M5OutputlPath, "M5.ldi.xml")

	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	m5Data, err := ioutil.ReadFile(m5LDIPath)
	if err != nil {
		return fmt.Errorf("M5 LDI 파일 읽기 실패: %v", err)
	}

	var mainRoot, m5Root Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI 파일 살펴보기 실패: %v", err)
	}
	if err := xml.Unmarshal(m5Data, &m5Root); err != nil {
		return fmt.Errorf("M5 LDI 파일 살펴보기 실패: %v", err)
	}

	// 컴포넌트 이름 -> 여러 속성( m5 및 m5demo 포함)
	m5Map := make(map[string][]Property)
	for _, el := range m5Root.Items {
		m5Map[el.Name] = append(m5Map[el.Name], el.Property...)
	}

	for i, el := range mainRoot.Items {
		if props, ok := m5Map[el.Name]; ok {
			// 현재 존재하는 속성명을 수집한다
			existing := make(map[string]bool)
			for _, p := range el.Property {
				existing[p.Name] = true
			}

			// 존재하지 않는 속성 추가(m5 및 m5demo 포함)
			for _, p := range props {
				if !existing[p.Name] {
					mainRoot.Items[i].Property = append(mainRoot.Items[i].Property, p)
				}
			}
		}
	}

	out, err := xml.MarshalIndent(mainRoot, "  ", "    ")
	if err != nil {
		return fmt.Errorf("주 LDI 병합 출력 실패: %v", err)
	}

	header := []byte(xml.Header)
	if err := ioutil.WriteFile(mainLDIPath, append(header, out...), 0644); err != nil {
		return fmt.Errorf("주 LDI 파일을 다시 쓰는 데 실패했습니다.: %v", err)
	}

	fmt.Println("✅ M5 및 m5demo 지표 병합 성공")
	return nil
}

