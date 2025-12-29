package C_S_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// 외부에 노출되는 C-S 포트 정보
type CSPort struct {
	Name      string // 포트명(P Name="Name"에서 추출)
	BlockType string // Inport / Outport（Require → Inport, Provide → Outport）
	SID       string // 여기서는 값을 고정으로 "unknow"로 설정합니다.
	PortType  string // 값을 "C-S"로 고정합니다.
}

// 내부 XML 구조
type xmlP struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

type xmlRequireFunction struct {
	Ps []xmlP `xml:"P"`
}

type xmlProvideFunction struct {
	Ps []xmlP `xml:"P"`
}

type xmlGraphicalInterface struct {
	Requires []xmlRequireFunction `xml:"RequireFunction"`
	Provides []xmlProvideFunction `xml:"ProvideFunction"`
}

// BuildDir<modelName>\simulink\graphicalInterface.xml에서 C-S 포트를 파싱합니다.
func GetCSPorts(modelName string) ([]CSPort, error) {
	var result []CSPort

	if M1_Public_Data.BuildDir == "" || modelName == "" {
		return result, nil
	}

	// BuildDir\<Model>\simulink\graphicalInterface.xml
	giPath := filepath.Join(M1_Public_Data.BuildDir, modelName, "simulink", "graphicalInterface.xml")

	data, err := os.ReadFile(giPath)
	if err != nil {
		// 파일이 없거나 읽기에 실패하면, 오류 정보를 포함하되 빈 리스트를 반환하며 경고 출력 여부는 호출 측에서 결정합니다.
		return result, fmt.Errorf("graphicalInterface.xml 읽기 실패 [%s]: %w", giPath, err)
	}

	var gi xmlGraphicalInterface
	if err := xml.Unmarshal(data, &gi); err != nil {
		return result, fmt.Errorf("graphicalInterface.xml 파싱 실패 [%s]: %w", giPath, err)
	}

	// RequireFunction → Inport로 간주합니다.
	for _, rf := range gi.Requires {
		name := ""
		for _, p := range rf.Ps {
			if p.Name == "Name" {
				name = normalizeName(p.Value)
				break
			}
		}
		if name == "" {
			continue
		}
		result = append(result, CSPort{
			Name:      name,
			BlockType: "Inport",
			SID:       "unknow",
			PortType:  "C-S",
		})
	}

	// ProvideFunction → Outport로 간주합니다.
	for _, pf := range gi.Provides {
		name := ""
		for _, p := range pf.Ps {
			if p.Name == "Name" {
				name = normalizeName(p.Value)
				break
			}
		}
		if name == "" {
			continue
		}
		result = append(result, CSPort{
			Name:      name,
			BlockType: "Outport",
			SID:       "unknow",
			PortType:  "C-S",
		})
	}

	return result, nil
}

// 이름에 포함된 줄바꿈/불필요한 공백을 하나의 공백으로 정규화합니다.
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}
