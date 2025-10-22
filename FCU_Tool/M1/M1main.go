package M1main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"FCU_Tools/M1/C_S_Analysis"
	"FCU_Tools/M1/File_Utils_M1"
	"FCU_Tools/M1/LDI_M1_Create"
	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/System_Analysis"
	"FCU_Tools/LDI_Create"     
	"FCU_Tools/Public_data"   
)

func M1_main() {
	// 1) 设置工作目录
	M1_Public_Data.SetWorkDir()

	// 2) 初始化目录
	File_Utils_M1.CreateDirectories()

	// 3) 读取 Windows 根路径
	reader := bufio.NewReader(os.Stdin)
	var srcRoot string
	for {
		fmt.Print("请输入一个 Windows 路径（该路径下一层是若干子文件夹）：")
		line, _ := reader.ReadString('\n')
		srcRoot = strings.TrimSpace(strings.Trim(line, `"`))
		if srcRoot != "" {
			break
		}
		fmt.Println("⚠️ 路径不能为空，请重新输入。")
	}

	// 4) 复制符合规则的 .slx 到 build/
	File_Utils_M1.CopyMatchingSLXFiles(srcRoot)

	// 5) 解压 build 目录中的 .slx
	File_Utils_M1.UnzipSLXsInBuildDir()

	// 6) 解析 system_root.xml（会把顶层系统集中存到 public_data.Systems）
	System_Analysis.AnalyzeSystems("root")

	// 7) 解析并合并 C-S 接口（把 C-S 端口挂到对应模型的顶层系统节点上）
	C_S_Analysis.AnalyzeCSPorts()

	// 8) 统一输出到控制台
	//M1_Public_Data.PrintAll()

	// 9) 导出到 TxtDir/<Model>.txt（每个模型一个文件）
	M1_Public_Data.ExportTreesToTxt()

	// 10) 基于 txt 计算 M1 并生成 LDI（每个模型一个 LDI）
	if err := LDI_M1_Create.GenerateM1AndLDIFromTxt(); err != nil {
		// 不 panic，打印错误后继续后续流程（如果有）
		fmt.Println("❌ 生成 M1 与 LDI 失败：", err)
	}

	// 11) 兜底：确保主 LDI 存在（合并目标）
	mainLDI := filepath.Join(Public_data.OutputDir, "result.ldi.xml")
	if _, err := os.Stat(mainLDI); os.IsNotExist(err) {
		if err := os.WriteFile(mainLDI, []byte("<ldi>\n</ldi>\n"), 0644); err != nil {
			fmt.Println("❌ 初始化主 LDI 失败：", err)
			return
		}
	}

	// 12) 合并：把 M1 的 <model>.ldi.xml 全部并到主 LDI（会带上 <property name="coverage.m1">…</property>）
	if err := LDI_Create.MergeAllFromM1LDIFolder(); err != nil {
		fmt.Println("❌ 合并主 LDI 失败：", err)
	} else {
		// ✅ 和 M2 同风格的成功提示
		fmt.Println("✅ M1지표 병합 성공")
		// 如果想顺便告诉用户主文件位置，可取消下一行注释：
		// fmt.Println("↳ 主 LDI 文件：", mainLDI)
	}
}
