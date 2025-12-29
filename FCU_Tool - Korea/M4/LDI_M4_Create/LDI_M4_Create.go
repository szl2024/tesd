package LDI_M4_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"FCU_Tools/Public_data"
)

// MergeM4ToMainLDI M4.ldi.xml의 coverage.m4 및 coverage.m4demo 지표를
// 주 LDI 파일 result.ldi.xml에 병합한다.
//
// 프로세스:
//   1) 주 LDI 파일(OutputDir/result.ldi.xml)과 M4/output/M4.ldi.xml을 읽는다.  
//   2) XML을 파싱하여 m4Map[name] → []Property를 구성한다.  
//   3) 주 LDI 요소를 순회하면서: 컴포넌트가 m4Map에 있으면 기존 속성을 확인하고, 누락된 속성은 추가한다.  
//   4) XML을 다시 직렬화하여 주 LDI 파일에 덮어쓴다.   
func MergeM4ToMainLDI() error {
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
	m4LDIPath := filepath.Join(Public_data.M4OutputlPath, "M4.ldi.xml")

	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	m4Data, err := ioutil.ReadFile(m4LDIPath)
	if err != nil {
		return fmt.Errorf("M4 LDI 파일 읽기 실패: %v", err)
	}

	var mainRoot, m4Root Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI XML 살펴보기 실패: %v", err)
	}
	if err := xml.Unmarshal(m4Data, &m4Root); err != nil {
		return fmt.Errorf("M4 LDI XML 살펴보기 실패: %v", err)
	}

	// 구성: 요소명 -> []Property
	m4Map := make(map[string][]Property)
	for _, el := range m4Root.Items {
		m4Map[el.Name] = append(m4Map[el.Name], el.Property...)
	}

	for i, el := range mainRoot.Items {
		if newProps, ok := m4Map[el.Name]; ok {
			// 기존 속성 집합을 구축하여 중복을 피한다.
			existing := make(map[string]bool)
			for _, p := range el.Property {
				existing[p.Name] = true
			}

			// 존재하지 않는 M4 속성( m4demo 포함)을 추가했습니다.
			for _, np := range newProps {
				if !existing[np.Name] {
					mainRoot.Items[i].Property = append(mainRoot.Items[i].Property, np)
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
	fmt.Println("✅ M4 및 m4demo 지표 병합 성공")
	return nil
}
