package LDI_M6_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"FCU_Tools/Public_data"
)

// MergeM6ToMainLDI M6.ldi.xml의 속성을 주 LDI 파일 result.ldi.xml에 병합한다.
//
// 프로세스:
//   1) 주 LDI(OutputDir/result.ldi.xml)와 M6 LDI(M6/output/M6.ldi.xml)를 읽는다.  
//   2) XML을 파싱하여 m6Map[name] → []Property를 구성한다 (coverage.m6 및 coverage.m6demo 포함).  
//   3) 주 LDI 요소를 순회하면서: 컴포넌트가 m6Map에 있으면 기존 속성을 확인하고, 누락된 속성을 추가한다.  
//   4) XML을 다시 직렬화하여 result.ldi.xml에 덮어쓴다.  
func MergeM6ToMainLDI() error {
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
	m6LDIPath := filepath.Join(Public_data.M6OutputlPath, "M6.ldi.xml")

	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	m6Data, err := ioutil.ReadFile(m6LDIPath)
	if err != nil {
		return fmt.Errorf("M6 LDI 파일 읽기 실패: %v", err)
	}

	var mainRoot, m6Root Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI 파일 살펴보기 실패: %v", err)
	}
	if err := xml.Unmarshal(m6Data, &m6Root); err != nil {
		return fmt.Errorf("M6 LDI 파일 살펴보기 실패: %v", err)
	}

	// 구성 요소 이름 -> 속성 목록( m6 + m6demo 지원)
	m6Map := make(map[string][]Property)
	for _, el := range m6Root.Items {
		m6Map[el.Name] = append(m6Map[el.Name], el.Property...)
	}

	for i, el := range mainRoot.Items {
		if props, ok := m6Map[el.Name]; ok {
			// 현재 존재하는 속성명을 수집하여 중복 추가를 방지하세요.
			existing := make(map[string]bool)
			for _, p := range el.Property {
				existing[p.Name] = true
			}

			// 모든 M6/M6demo 속성을 병합합니다.
			for _, newProp := range props {
				if !existing[newProp.Name] {
					mainRoot.Items[i].Property = append(mainRoot.Items[i].Property, newProp)
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
		return fmt.Errorf("주 LDI 파일을 다시 쓰는 데 실패했습니다: %v", err)
	}

	fmt.Println("✅ M6 및 m6demo 지표 병합 성공")
	return nil
}

