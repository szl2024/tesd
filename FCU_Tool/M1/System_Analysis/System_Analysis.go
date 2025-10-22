package System_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/Port_Analysis"
	"FCU_Tools/M1/SubSystem_Analysis"
)

// ---- 日志开关：置为 true 可查看解析详情；false 静默（仅异常/告警显示） ----
const verboseSys = false

func vprintln(a ...any) {
	if verboseSys {
		fmt.Println(a...)
	}
}

func vprintf(format string, a ...any) {
	if verboseSys {
		fmt.Printf(format, a...)
	}
}

// ========= 结构 A：<System><Block> =========
type XMLBlockA struct {
	BlockType string `xml:"BlockType,attr"`
	Name      string `xml:"Name,attr"`
	SID       string `xml:"SID,attr"`
	InnerXML  string `xml:",innerxml"`
}
type XMLRootA struct {
	Blocks []XMLBlockA `xml:"System>Block"`
}

// ========= 结构 B：根直接含 <Block> =========
type XMLBlockB struct {
	BlockType string `xml:"BlockType,attr"`
	Name      string `xml:"Name,attr"`
	SID       string `xml:"SID,attr"`
	InnerXML  string `xml:",innerxml"`
}
type XMLRootB struct {
	Blocks []XMLBlockB `xml:"Block"`
}

// AnalyzeSystems 扫描 BuildDir 下的 system_<fileTag>.xml，构造顶层系统树并集中保存
func AnalyzeSystems(fileTag string) {
	rootDir := M1_Public_Data.BuildDir
	if rootDir == "" {
		fmt.Println("❌ BuildDir 未设置")
		return
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		fmt.Println("❌ 无法读取 BuildDir：", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelName := entry.Name()
		targetFile := fmt.Sprintf("system_%s.xml", fileTag)
		xmlPath := filepath.Join(rootDir, modelName, "simulink", "systems", targetFile)

		if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
			fmt.Printf("⚠️ 模型 %s 未找到 %s，跳过\n", modelName, targetFile)
			continue
		}
		vprintln("............................................................")
		vprintf("🔍 解析模型：%s (%s)\n", modelName, targetFile)

		data, err := os.ReadFile(xmlPath)
		if err != nil {
			fmt.Printf("❌ 无法读取 %s：%v\n", xmlPath, err)
			continue
		}

		// 1) 优先按 <System><Block> 结构解析
		var rootA XMLRootA
		_ = xml.Unmarshal(data, &rootA)

		blocksFound := false
		if len(rootA.Blocks) > 0 {
			blocksFound = true
			consumeBlocksA(modelName, data, rootA.Blocks)
		}

		// 2) 回退为根直接 <Block> 结构
		if !blocksFound {
			var rootB XMLRootB
			if err := xml.Unmarshal(data, &rootB); err == nil && len(rootB.Blocks) > 0 {
				blocksFound = true
				consumeBlocksB(modelName, data, rootB.Blocks)
			}
		}

		if !blocksFound {
			fmt.Printf("⚠️ 未在 %s 中找到任何 Block，可能 XML 结构不同，请检查。\n", xmlPath)
		}
	}

	vprintf("✅ 所有模型 %s 分析完成。\n", fileTag)
}

func consumeBlocksA(modelName string, data []byte, blocks []XMLBlockA) {
	ports, _ := Port_Analysis.ParsePortsInXML(string(data))

	for _, block := range blocks {
		if block.BlockType != "SubSystem" {
			continue
		}
		if isPseudoSubSystem(block.InnerXML) {
			continue
		}

		sys := &M1_Public_Data.SystemInfo{
			Model: modelName,
			Name:  block.Name,
			SID:   block.SID,
			Port:  ports,
		}

		vprintf("✅ 检测到有效 SubSystem：Name=%s, SID=%s (Ports=%d)\n",
			sys.Name, sys.SID, len(sys.Port))
		for _, p := range sys.Port {
			vprintf("    ↳ Port: Name=%s, SID=%s, Type=%s, IO=%s\n",
				p.Name, p.SID, p.Type, p.IO)
		}

		// 递归分析子系统（结果挂在 sys.SubSystem）
		vprintf("🚀 开始分析子系统：system_%s.xml\n", sys.SID)
		SubSystem_Analysis.AnalyzeSubSystems(modelName, sys.SID, sys)

		// 集中保存
		M1_Public_Data.AddTopSystem(sys)
	}
}

func consumeBlocksB(modelName string, data []byte, blocks []XMLBlockB) {
	ports, _ := Port_Analysis.ParsePortsInXML(string(data))

	for _, block := range blocks {
		if block.BlockType != "SubSystem" {
			continue
		}
		if isPseudoSubSystem(block.InnerXML) {
			continue
		}

		sys := &M1_Public_Data.SystemInfo{
			Model: modelName,
			Name:  block.Name,
			SID:   block.SID,
			Port:  ports,
		}

		vprintf("✅ 检测到有效 SubSystem：Name=%s, SID=%s (Ports=%d)\n",
			sys.Name, sys.SID, len(sys.Port))
		for _, p := range sys.Port {
			vprintf("    ↳ Port: Name=%s, SID=%s, Type=%s, IO=%s\n",
				p.Name, p.SID, p.Type, p.IO)
		}

		vprintf("🚀 开始分析子系统：system_%s.xml\n", sys.SID)
		SubSystem_Analysis.AnalyzeSubSystems(modelName, sys.SID, sys)

		M1_Public_Data.AddTopSystem(sys)
	}
}

// 判断“假性子系统”（无端口等）
func isPseudoSubSystem(inner string) bool {
	s := strings.ReplaceAll(inner, " ", "")
	s = strings.ToLower(s)

	if strings.Contains(s, "<portcounts/>") || strings.Contains(s, "<portcounts></portcounts>") {
		return true
	}
	if strings.Contains(s, `<pname="ports">[]</p>`) || strings.Contains(s, `<pname="ports"></p>`) {
		return true
	}
	return false
}
