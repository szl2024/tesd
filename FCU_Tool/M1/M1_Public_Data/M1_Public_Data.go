package M1_Public_Data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ========== 日志开关：仅错误打印 ==========
// 置为 true 可恢复所有普通信息打印；默认 false 静默普通信息。
const verboseCore = false

// Dir 保存当前 M1 的工作目录
var Dir string

const (
	PortIn  = "IN"  // 输入端口
	PortOut = "OUT" // 输出端口
)

// 各输出目录（由其它模块写入/使用）
var (
	BuildDir  string
	OutputDir string
	LdiDir    string
	TxtDir    string
)

// ✅ 全局集中存储：所有已解析的顶层系统（每个元素是一棵系统树）
var Systems []*SystemInfo

// SystemInfo 为系统/子系统节点
type SystemInfo struct {
	Model     string         // ✅ 所属模型名（用于把 C-S 端口挂到对应系统）
	Name      string
	SID       string
	SubSystem []*SystemInfo
	Port      []*PortInfo
}

// PortInfo 为端口信息
type PortInfo struct {
	Name string
	SID  string
	Type string // 普通 S-R；C-S 接口为 C-S
	IO   string // IN / OUT
}

// ---------- 基础工具 ----------

func logMsg(prefix, msg string) {
	// 仅当是严重错误（❌）或开启了 verbose 时才打印
	if prefix == "❌" || verboseCore {
		fmt.Println(prefix, msg)
	}
}

// SetWorkDir 设置当前工作目录（内部处理错误与输出）
func SetWorkDir() {
	wd, err := os.Getwd()
	if err != nil {
		logMsg("❌", "获取工作目录失败："+err.Error())
		return
	}
	abs, _ := filepath.Abs(wd)
	Dir = abs
	// 成功信息静默（只有 verboseCore=true 才会打印）
	logMsg("✅", "当前工作目录："+Dir)
}

// ---------- 全局存取工具 ----------

// ResetAll 清空集中存储
func ResetAll() { Systems = nil }

// AddTopSystem 增加一个顶层系统节点（整棵树的根）
func AddTopSystem(sys *SystemInfo) { Systems = append(Systems, sys) }

// AttachCSPortsToModel 为指定模型的所有顶层系统合并一批 C-S 端口
func AttachCSPortsToModel(model string, ports []*PortInfo) {
	if len(ports) == 0 {
		return
	}
	for _, sys := range Systems {
		if sys.Model == model {
			sys.Port = append(sys.Port, ports...)
		}
	}
}

// ---------- 控制台输出（保留：用于最终展示） ----------

const treeSep = "........................................................................"

// PrintAll 统一输出 Systems 中的所有系统树（包含普通端口与 C-S 端口）
// 每颗树之间增加空行与分隔线，提升可读性。
func PrintAll() {
	for i, sys := range Systems {
		printSystem(sys, 0)
		// 额外的视觉分隔：空行 + 分隔线（最后一棵树后不再打印分隔线）
		if i < len(Systems)-1 {
			fmt.Println()
			fmt.Println(treeSep)
		}
	}
}

// 递归打印（控制台）
func printSystem(sys *SystemInfo, depth int) {
	prefix := strings.Repeat("    ", depth)
	fmt.Printf("%s✅ System: %s (Model=%s, SID=%s)\n", prefix, sys.Name, sys.Model, sys.SID)

	for _, p := range sys.Port {
		fmt.Printf("%s   ↳ Port: Name=%s, SID=%s, IO=%s, Type=%s\n",
			prefix, p.Name, p.SID, p.IO, p.Type)
	}
	for _, sub := range sys.SubSystem {
		printSystem(sub, depth+1)
	}
}

// ---------- 文本导出（新增） ----------

// ExportTreesToTxt
// 将 Systems 中的系统树，按模型名分文件导出到 TxtDir：<TxtDir>/<Model>.txt
// 对齐规则：在每个模型文件内按最大列宽竖向对齐；
// 端口行在标签前空 1 格；Port: 后仅 1 个空格；[L2 Port] 不输出 Type。
func ExportTreesToTxt() {
	if TxtDir == "" {
		if OutputDir != "" {
			TxtDir = filepath.Join(OutputDir, "txt")
		} else if Dir != "" {
			TxtDir = filepath.Join(Dir, "M1", "output", "txt")
		} else {
			logMsg("❌", "无法确定 TxtDir 的路径，请先调用 SetWorkDir() 与 CreateDirectories()")
			return
		}
	}
	if err := os.MkdirAll(TxtDir, 0o755); err != nil {
		logMsg("❌", "创建 TxtDir 失败："+err.Error())
		return
	}

	// 1) 按模型分组
	group := make(map[string][]*SystemInfo)
	order := make([]string, 0)
	for _, sys := range Systems {
		if sys == nil {
			continue
		}
		model := strings.TrimSpace(sys.Model)
		if model == "" {
			model = "unknown_model"
		}
		if _, exists := group[model]; !exists {
			order = append(order, model)
		}
		group[model] = append(group[model], sys)
	}

	// 2) 逐模型写文件（两遍：先统计最大宽度，再格式化输出）
	for _, model := range order {
		systems := group[model]

		// (2.1) 第一遍：统计最大宽度
		maxSysName := 0
		maxPortName, maxPortSID, maxPortIO, maxPortType := 0, 0, 0, 0

		var scan func(*SystemInfo)
		scan = func(s *SystemInfo) {
			if l := len(s.Name); l > maxSysName {
				maxSysName = l
			}
			for _, p := range s.Port {
				if l := len(p.Name); l > maxPortName {
					maxPortName = l
				}
				if l := len(p.SID); l > maxPortSID {
					maxPortSID = l
				}
				if l := len(p.IO); l > maxPortIO {
					maxPortIO = l
				}
				if l := len(p.Type); l > maxPortType {
					maxPortType = l
				}
			}
			for _, sub := range s.SubSystem {
				scan(sub)
			}
		}
		for _, s := range systems {
			scan(s)
		}
		// 下限保护
		if maxSysName < 1 { maxSysName = 1 }
		if maxPortName < 1 { maxPortName = 1 }
		if maxPortSID  < 1 { maxPortSID  = 1 }
		if maxPortIO   < 1 { maxPortIO   = 1 }
		if maxPortType < 1 { maxPortType = 1 }

		// (2.2) 第二遍：格式化输出
		var b strings.Builder

		// System 行：对齐 Name；括号内不等宽，避免 “SID=4  )”
		sysFmt   := "%s System: %-*s (Model=%s, SID=%s)\n"
		// L1 端口（含 Type）
		portFmtL1 := "%s Port: Name=%-*s, SID=%-*s, IO=%-*s, Type=%-*s\n"
		// L2 端口（不含 Type）
		portFmtL2 := "%s Port: Name=%-*s, SID=%-*s, IO=%-*s\n"

		var writeWalk func(*SystemInfo, int)
		writeWalk = func(s *SystemInfo, depth int) {
			levelTag := "[L1]"
			portTag  := "[L1 Port]"
			if depth >= 1 {
				levelTag = "[L2]"
				portTag  = "[L2 Port]"
			}

			// System 行（标签前不加空格）
			b.WriteString(fmt.Sprintf(
				sysFmt,
				levelTag, maxSysName, s.Name, s.Model, s.SID,
			))

			// Port 行（在标签前空 1 格；Port: 后仅 1 空格）
			for _, p := range s.Port {
				if depth >= 1 {
					// [L2 Port]：不输出 Type
					b.WriteString(fmt.Sprintf(
						portFmtL2,
						"	"+portTag, // 在这里空一格再打印标签
						maxPortName, p.Name,
						maxPortSID,  p.SID,
						maxPortIO,   p.IO,
					))
				} else {
					// [L1 Port]：保留 Type
					b.WriteString(fmt.Sprintf(
						portFmtL1,
						"	"+portTag, // 在这里空一格再打印标签
						maxPortName, p.Name,
						maxPortSID,  p.SID,
						maxPortIO,   p.IO,
						maxPortType, p.Type,
					))
				}
			}

			for _, sub := range s.SubSystem {
				writeWalk(sub, depth+1)
			}
		}

		for i, s := range systems {
			writeWalk(s, 0)
			if i < len(systems)-1 {
				b.WriteString("\n")
				b.WriteString(treeSep)
				b.WriteString("\n")
			}
		}

		filename := sanitizeFilename(model) + ".txt"
		fullpath := filepath.Join(TxtDir, filename)
		if err := os.WriteFile(fullpath, []byte(b.String()), 0o644); err != nil {
			logMsg("❌", fmt.Sprintf("写出 txt 失败：%s：%v", fullpath, err))
			continue
		}
	}
}



// formatSystem 将系统树格式化为文本（与控制台输出风格一致）
func formatSystem(b *strings.Builder, sys *SystemInfo, depth int) {
	prefix := strings.Repeat("    ", depth)
	b.WriteString(fmt.Sprintf("%s✅ System: %s (Model=%s, SID=%s)\n", prefix, sys.Name, sys.Model, sys.SID))
	for _, p := range sys.Port {
		b.WriteString(fmt.Sprintf("%s   ↳ Port: Name=%s, SID=%s, IO=%s, Type=%s\n",
			prefix, p.Name, p.SID, p.IO, p.Type))
	}
	for _, sub := range sys.SubSystem {
		formatSystem(b, sub, depth+1)
	}
}

// sanitizeFilename 处理 Windows/Unix 非法文件名字符
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed"
	}
	// Windows 非法字符：<>:"/\|?*
	illegal := []string{`<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`}
	for _, ch := range illegal {
		name = strings.ReplaceAll(name, ch, "_")
	}
	// 额外清理不可见字符
	name = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, name)
	return name
}
