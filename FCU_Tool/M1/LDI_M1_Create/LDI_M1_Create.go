package LDI_M1_Create

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// 汇总细分计数，用于打印“统计摘要”
type metricsSummary struct {
	L1PortTotal int
	L1IN        int
	L1OUT       int
	L1TypeCS    int
	L1TypeSR    int

	L2Total     int // == class
	L2PortTotal int
	L2IN        int
	L2OUT       int
}

// GenerateM1AndLDIFromTxt
// 读取 Public_data.TxtDir 下的每个 <Model>.txt：
// 1) 只取“第一棵根系统”计算：
//    portasr = 根端口计数（S-R=1.0，C-S=1.2）
//    class   = 根的直系子系统数量（深度=1）
//    portsim = 所有直系子系统（深度=1）的端口总数
//    M1 = portasr * class * portsim
// 2) 把“统计摘要”与“M1 结果”追加写回该 txt 尾部；
// 3) 在 Public_data.LdiDir 下生成 <Model>.ldi.xml：
//    <ldi><element name="模型名"><property name="coverage.m1">M1</property></element></ldi>
func GenerateM1AndLDIFromTxt() error {
	txtDir := M1_Public_Data.TxtDir
	ldiDir := M1_Public_Data.LdiDir

	if txtDir == "" {
		return fmt.Errorf("TxtDir 未设置，请先调用 CreateDirectories 和 ExportTreesToTxt")
	}
	if ldiDir == "" {
		// 兜底设置
		if M1_Public_Data.OutputDir != "" {
			ldiDir = filepath.Join(M1_Public_Data.OutputDir, "LDI")
			M1_Public_Data.LdiDir = ldiDir
		} else if M1_Public_Data.Dir != "" {
			ldiDir = filepath.Join(M1_Public_Data.Dir, "M1", "output", "LDI")
			M1_Public_Data.LdiDir = ldiDir
		} else {
			return fmt.Errorf("无法确定 LdiDir 路径")
		}
	}
	if err := os.MkdirAll(ldiDir, 0o755); err != nil {
		return fmt.Errorf("创建 LdiDir 失败: %w", err)
	}

	ents, err := os.ReadDir(txtDir)
	if err != nil {
		return fmt.Errorf("读取 TxtDir 失败: %w", err)
	}

	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			continue
		}

		txtPath := filepath.Join(txtDir, e.Name())
		model := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))

		portasr, class, portsim, rootModel, summary, err := parseRootAndMetrics(txtPath)
		if err != nil {
			fmt.Printf("⚠️ 解析失败（%s）：%v\n", txtPath, err)
			continue
		}
		if rootModel == "" {
			rootModel = model
		}

		m1 := portasr * float64(class) * float64(portsim)

		// 1) 先写“统计摘要”，再写“M1 结果”
		if err := appendM1ResultToTxt(txtPath, summary, portasr, class, portsim, m1); err != nil {
			fmt.Printf("⚠️ 写回 M1 结果失败（%s）：%v\n", txtPath, err)
		}

		// 2) 生成 LDI
		ldiPath := filepath.Join(ldiDir, sanitizeFilename(rootModel)+".ldi.xml")
		if err := writeLDI(ldiPath, rootModel, m1); err != nil {
			fmt.Printf("⚠️ 生成 LDI 失败（%s）：%v\n", ldiPath, err)
			continue
		}
	}
	return nil
}

// 解析“第一棵根系统”的三个指标：portasr、class、portsim，并返回根的模型名与“统计摘要”
func parseRootAndMetrics(txtPath string) (portasr float64, class int, portsim int, rootModel string, summary metricsSummary, err error) {
	f, err := os.Open(txtPath)
	if err != nil {
		return 0, 0, 0, "", metricsSummary{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	inFirstTree := false // 是否已进入第一棵根系统块
	started := false     // 是否已经遇到第一条 [L1] System:

	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimLeft(line, " \t") // 容忍行首空格与 \t（端口标签前可能多一格）

		tag := detectTag(trim) // "[L1]" | "[L1 Port]" | "[L2]" | "[L2 Port]" | ""
		switch tag {

		case "[L1]":
			// 只把带 System: 的 [L1] 行视为根系统起点
			if hasWordAfterTag(trim, "System:") {
				if started {
					goto DONE // 第二棵根系统出现，结束统计
				}
				started = true
				inFirstTree = true
				rootModel = parseModelFromSystemLine(trim) // (Model=...)
			}

		case "[L1 Port]":
			if !inFirstTree {
				continue
			}
			// 细分计数
			summary.L1PortTotal++
			if parsePortType(trim) == "C-S" {
				summary.L1TypeCS++
				portasr += 1.2 // 权重
			} else {
				summary.L1TypeSR++
				portasr += 1.0
			}
			switch parsePortIO(trim) {
			case "IN":
				summary.L1IN++
			case "OUT":
				summary.L1OUT++
			}

		case "[L2]":
			if !inFirstTree {
				continue
			}
			// 只统计“系统行”的 [L2]，避免把 [L2 Port] 误算入 class
			if hasWordAfterTag(trim, "System:") {
				class++
				summary.L2Total++
			}

		case "[L2 Port]":
			if !inFirstTree {
				continue
			}
			portsim++
			summary.L2PortTotal++
			switch parsePortIO(trim) {
			case "IN":
				summary.L2IN++
			case "OUT":
				summary.L2OUT++
			}
		}
	}

DONE:
	if err := sc.Err(); err != nil {
		return portasr, class, portsim, rootModel, summary, err
	}
	return portasr, class, portsim, rootModel, summary, nil
}

// 检测行首标签（去掉左侧空白后）
func detectTag(trimmed string) string {
	switch {
	case strings.HasPrefix(trimmed, "[L1]"):
		return "[L1]"
	case strings.HasPrefix(trimmed, "[L1 Port]"):
		return "[L1 Port]"
	case strings.HasPrefix(trimmed, "[L2]"):
		return "[L2]"
	case strings.HasPrefix(trimmed, "[L2 Port]"):
		return "[L2 Port]"
	default:
		return ""
	}
}

// 判断在标签之后是否出现了某个词（如 "System:" 或 "Port:"），容忍标签和词之间的任意空格
func hasWordAfterTag(trimmed, word string) bool {
	// 找到第一个 ']' 之后的子串
	i := strings.Index(trimmed, "]")
	if i < 0 || i+1 >= len(trimmed) {
		return false
	}
	rest := strings.TrimLeft(trimmed[i+1:], " \t")
	return strings.HasPrefix(rest, word)
}

// 从端口行中抽取 Type，适配当前导出格式："... Type=C-S"
func parsePortType(line string) string {
	const key = "Type="
	i := strings.Index(line, key)
	if i == -1 {
		return ""
	}
	rest := strings.TrimSpace(line[i+len(key):])
	// 取到下一个空白或逗号/右括号为止（通常 Type 在行尾）
	for idx, r := range rest {
		if r == ' ' || r == '\t' || r == ',' || r == ')' {
			return rest[:idx]
		}
	}
	return rest
}

// 从端口行中抽取 IO，适配格式："..., IO=IN , ..." 或 "..., IO=OUT, ..."
func parsePortIO(line string) string {
	const key = "IO="
	i := strings.Index(line, key)
	if i == -1 {
		return ""
	}
	rest := strings.TrimSpace(line[i+len(key):])
	for idx, r := range rest {
		if r == ' ' || r == '\t' || r == ',' || r == ')' {
			return rest[:idx]
		}
	}
	return rest
}

func parseModelFromSystemLine(line string) string {
	// 形如：System: Runnable_10ms_sys (Model=TurnLightAct, SID=4)
	open := strings.Index(line, "(Model=")
	if open < 0 {
		return ""
	}
	rest := line[open+len("(Model="):]
	end := strings.IndexAny(rest, ",)")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func appendM1ResultToTxt(txtPath string, sum metricsSummary, portasr float64, class, portsim int, m1 float64) error {
	f, err := os.OpenFile(txtPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var sb strings.Builder
	sb.WriteString("\n")
	// 先输出统计摘要
	sb.WriteString("----- 统计摘要 --------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("[L1 Port] 总数 = %d（IN=%d, OUT=%d；Type：C-S=%d, S-R=%d）\n",
		sum.L1PortTotal, sum.L1IN, sum.L1OUT, sum.L1TypeCS, sum.L1TypeSR))
	sb.WriteString(fmt.Sprintf("[L2]      总数 = %d\n", sum.L2Total))
	sb.WriteString(fmt.Sprintf("[L2 Port] 总数 = %d（IN=%d, OUT=%d）\n",
		sum.L2PortTotal, sum.L2IN, sum.L2OUT))

	// 再输出 M1 结果
	sb.WriteString("----- M1 结果 --------------------------------------------------\n")
	sb.WriteString("portasr = ")
	sb.WriteString(fmtFloat(portasr))
	sb.WriteString(", class = ")
	sb.WriteString(strconv.Itoa(class))
	sb.WriteString(", portsim = ")
	sb.WriteString(strconv.Itoa(portsim))
	sb.WriteString("\nM1 = portasr * class * portsim = ")
	sb.WriteString(fmtFloat(m1))
	sb.WriteString("\n")

	_, err = f.WriteString(sb.String())
	return err
}

func writeLDI(ldiPath, model string, m1 float64) error {
	if err := os.MkdirAll(filepath.Dir(ldiPath), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(
		"<ldi>\n  <element name=\"%s\">\n    <property name=\"coverage.m1\">%s</property>\n  </element>\n</ldi>\n",
		escapeXML(model), fmtFloat(m1),
	)
	return os.WriteFile(ldiPath, []byte(content), 0o644)
}

func fmtFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed"
	}
	illegal := []string{`<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`}
	for _, ch := range illegal {
		name = strings.ReplaceAll(name, ch, "_")
	}
	name = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, name)
	return name
}

func escapeXML(s string) string {
	repl := []struct{ old, new string }{
		{"&", "&amp;"},
		{"<", "&lt;"},
		{">", "&gt;"},
		{`"`, "&quot;"},
		{"'", "&apos;"},
	}
	for _, r := range repl {
		s = strings.ReplaceAll(s, r.old, r.new)
	}
	return s
}
