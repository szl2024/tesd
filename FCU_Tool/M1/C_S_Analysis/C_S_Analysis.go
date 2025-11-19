package C_S_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// 对外暴露的 C-S 端口信息
type CSPort struct {
	Name      string // 端口名（从 P Name="Name" 里取）
	BlockType string // Inport / Outport（Require → Inport, Provide → Outport）
	SID       string // 这里固定 "unknow"
	PortType  string // 固定 "C-S"
}

// 内部 XML 结构
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

// 从 BuildDir\<modelName>\simulink\graphicalInterface.xml 中解析 C-S 端口
func GetCSPorts(modelName string) ([]CSPort, error) {
	var result []CSPort

	if M1_Public_Data.BuildDir == "" || modelName == "" {
		return result, nil
	}

	// BuildDir\<Model>\simulink\graphicalInterface.xml
	giPath := filepath.Join(M1_Public_Data.BuildDir, modelName, "simulink", "graphicalInterface.xml")

	data, err := os.ReadFile(giPath)
	if err != nil {
		// 如果文件不存在或读取失败，这里返回空列表但带错误信息，由调用方决定是否打印告警
		return result, fmt.Errorf("读取 graphicalInterface.xml 失败 [%s]: %w", giPath, err)
	}

	var gi xmlGraphicalInterface
	if err := xml.Unmarshal(data, &gi); err != nil {
		return result, fmt.Errorf("解析 graphicalInterface.xml 失败 [%s]: %w", giPath, err)
	}

	// RequireFunction → 视为 Inport
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

	// ProvideFunction → 视为 Outport
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

// 把名字里的换行 / 多余空白压成一个空格
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}
