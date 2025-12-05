package LDI_M3_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"FCU_Tools/Public_data"
)

// MergeM3ToMainLDI M3.ldi.xml의 속성을 주 LDI 파일 result.ldi.xml에 병합한다.
//
// 프로세스:
//   1) 주 LDI 파일(OutputDir/result.ldi.xml)과 M3 LDI 파일(M3/output/M3.ldi.xml)을 읽는다.  
//   2) Root{[]Element}로 파싱한다.  
//   3) m3Map[name] → []Property를 구성한다 (즉, coverage.m3 / coverage.m3demo).  
//   4) 주 LDI 요소를 순회하면서: 컴포넌트가 m3Map에 있으면 기존 속성을 확인하고, 없으면 추가한다.  
//   5) 다시 직렬화하여 result.ldi.xml에 덮어쓴다.  

func MergeM3ToMainLDI() error {
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
	m3LDIPath := filepath.Join(Public_data.M3OutputlPath, "M3.ldi.xml")

	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	m3Data, err := ioutil.ReadFile(m3LDIPath)
	if err != nil {
		return fmt.Errorf("M3 LDI 파일 읽기 실패: %v", err)
	}

	var mainRoot, m3Root Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI 파일 살펴보기 실패: %v", err)
	}
	if err := xml.Unmarshal(m3Data, &m3Root); err != nil {
		return fmt.Errorf("M3 LDI 파일 살펴보기 실패: %v", err)
	}

	m3Map := make(map[string][]Property)
	for _, el := range m3Root.Items {
		m3Map[el.Name] = append(m3Map[el.Name], el.Property...)
	}

	for i, el := range mainRoot.Items {
		if props, ok := m3Map[el.Name]; ok {

			existing := make(map[string]bool)
			for _, p := range el.Property {
				existing[p.Name] = true
			}

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
		return fmt.Errorf("주 LDI 파일을 다시 쓰는 데 실패했습니다.: %v", err)
	}

	fmt.Println("✅ M3 및 m3demo 지표 병합 성공.")
	return nil
}
