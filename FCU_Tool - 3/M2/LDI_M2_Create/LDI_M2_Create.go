package LDI_M2_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"FCU_Tools/Public_data"
)

// MergeM2ToMainLDI는 M2.ldi.xml의 coverage.m2 지표를
// 메인 LDI 파일 result.ldi.xml에 병합한다.
//
// 프로세스:
//   1) 메인 LDI 파일(OutputDir/result.ldi.xml)과
//      M2 LDI 파일(M2/output/M2.ldi.xml)을 읽는다.
//   2) Root{[]Element} 구조체로 파싱한다.
//   3) M2 요소를 순회하며 m2Map[name] = coverage.m2 값을 구축한다.
//   4) 메인 LDI 요소를 순회: name이 m2Map에 존재하고
//      아직 coverage.m2 속성이 없으면 property를 추가한다.
//   5) XML을 다시 직렬화하여 메인 LDI 파일에 덮어쓴다.

func MergeM2ToMainLDI() error {
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
	m2LDIPath := filepath.Join(Public_data.M2OutputlPath, "M2.ldi.xml")

	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	m2Data, err := ioutil.ReadFile(m2LDIPath)
	if err != nil {
		return fmt.Errorf("M2 LDI 파일 읽기 실패: %v", err)
	}

	var mainRoot, m2Root Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("주 LDI XML 읽기 실패: %v", err)
	}
	if err := xml.Unmarshal(m2Data, &m2Root); err != nil {
		return fmt.Errorf("M2 LDI XML 읽기 실패: %v", err)
	}

	m2Map := make(map[string]string)
	for _, el := range m2Root.Items {
		for _, p := range el.Property {
			if p.Name == "coverage.m2" {
				m2Map[el.Name] = p.Value
			}
		}
	}

	for i, el := range mainRoot.Items {
		if val, ok := m2Map[el.Name]; ok {
			alreadyExists := false
			for _, p := range el.Property {
				if p.Name == "coverage.m2" {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				mainRoot.Items[i].Property = append(mainRoot.Items[i].Property, Property{
					Name:  "coverage.m2",
					Value: val,
				})

			}
		} else {

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
	fmt.Println("✅ M2지표 병합 성공")
	return nil
}
