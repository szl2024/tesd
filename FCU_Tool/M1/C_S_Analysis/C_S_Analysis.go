package C_S_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// ---- 日志开关：设为 true 可查看 C-S 解析详情；false 为静默（仅最终总输出显示） ----
const verboseCSPorts = false

func vprintln(a ...any) {
	if verboseCSPorts {
		fmt.Println(a...)
	}
}

func vprintf(format string, a ...any) {
	if verboseCSPorts {
		fmt.Printf(format, a...)
	}
}

// XML 结构（根据 graphicalInterface.xml 的常见结构做最小定义）
type XMLParam struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

type ProvideFunction struct {
	P []XMLParam `xml:"P"`
}

type RequireFunction struct {
	P []XMLParam `xml:"P"`
}

type GraphicalInterface struct {
	ProvideFuncs []ProvideFunction `xml:"ProvideFunction"`
	RequireFuncs []RequireFunction `xml:"RequireFunction"`
}

// AnalyzeCSPorts 逐模型解析 graphicalInterface.xml，并把 C-S 端口合并进对应模型的顶层系统节点
func AnalyzeCSPorts() {
	rootDir := M1_Public_Data.BuildDir
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		// 读取 BuildDir 失败属于异常，仍然需要提示
		fmt.Printf("❌ 无法读取 BuildDir: %v\n", err)
		return
	}

	vprintln("🚀 开始分析 graphicalInterface.xml 文件中的 C-S 函数接口...")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		modelName := e.Name()
		xmlPath := filepath.Join(rootDir, modelName, "simulink", "graphicalInterface.xml")

		if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
			continue // 忽略无文件模型
		}

		data, err := os.ReadFile(xmlPath)
		if err != nil {
			// 文件读失败也提示一下
			fmt.Printf("⚠️ 无法读取 %s: %v\n", xmlPath, err)
			continue
		}

		var gi GraphicalInterface
		if err := xml.Unmarshal(data, &gi); err != nil {
			// 解析失败提示
			fmt.Printf("⚠️ 无法解析 %s: %v\n", xmlPath, err)
			continue
		}

		// 收集 C-S 端口到切片，稍后统一合并
		var csPorts []*M1_Public_Data.PortInfo

		// ProvideFunction → OUT 端口
		for _, pf := range gi.ProvideFuncs {
			for _, p := range pf.P {
				if strings.EqualFold(p.Name, "Name") && strings.TrimSpace(p.Value) != "" {
					port := &M1_Public_Data.PortInfo{
						Name: p.Value,
						SID:  "unknown",
						Type: "C-S",
						IO:   "OUT",
					}
					csPorts = append(csPorts, port)
				}
			}
		}

		// RequireFunction → IN 端口
		for _, rf := range gi.RequireFuncs {
			for _, p := range rf.P {
				if strings.EqualFold(p.Name, "Name") && strings.TrimSpace(p.Value) != "" {
					port := &M1_Public_Data.PortInfo{
						Name: p.Value,
						SID:  "unknown",
						Type: "C-S",
						IO:   "IN",
					}
					csPorts = append(csPorts, port)
				}
			}
		}

		if len(csPorts) == 0 {
			continue
		}

		// 合并到该模型的所有顶层系统节点
		M1_Public_Data.AttachCSPortsToModel(modelName, csPorts)

		// 仅在 verbose 时打印模型与端口列表
		vprintf("✅ 模型：%s\n", modelName)
		for _, p := range csPorts {
			vprintf("   ↳ Port: Name=%s, IO=%s, Type=%s\n", p.Name, p.IO, p.Type)
		}
	}

	vprintln("🏁 C-S 函数接口分析完成。")
}

// 可选：单模型版本（需要时可调用）
func AnalyzeSingleModelCS(modelName string) []*M1_Public_Data.PortInfo {
	rootDir := M1_Public_Data.BuildDir
	xmlPath := filepath.Join(rootDir, modelName, "simulink", "graphicalInterface.xml")

	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		fmt.Printf("⚠️ 模型 %s 未找到 graphicalInterface.xml\n", modelName)
		return nil
	}

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		fmt.Printf("❌ 无法读取 %s: %v\n", xmlPath, err)
		return nil
	}

	var gi GraphicalInterface
	if err := xml.Unmarshal(data, &gi); err != nil {
		fmt.Printf("⚠️ 无法解析 %s: %v\n", xmlPath, err)
		return nil
	}

	var ports []*M1_Public_Data.PortInfo

	// ProvideFunction → OUT
	for _, pf := range gi.ProvideFuncs {
		for _, p := range pf.P {
			if p.Name == "Name" && strings.TrimSpace(p.Value) != "" {
				ports = append(ports, &M1_Public_Data.PortInfo{
					Name: p.Value, SID: "unknown", Type: "C-S", IO: "OUT",
				})
			}
		}
	}
	// RequireFunction → IN
	for _, rf := range gi.RequireFuncs {
		for _, p := range rf.P {
			if p.Name == "Name" && strings.TrimSpace(p.Value) != "" {
				ports = append(ports, &M1_Public_Data.PortInfo{
					Name: p.Value, SID: "unknown", Type: "C-S", IO: "IN",
				})
			}
		}
	}
	return ports
}
