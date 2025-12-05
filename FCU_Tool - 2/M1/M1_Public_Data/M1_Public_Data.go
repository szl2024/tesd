package M1_Public_Data

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	WorkDir   string
	M1Dir     string
	BuildDir  string
	OutputDir string
	LDIDir    string
	TxtDir    string

	SrcPath   string // ← 这里用来保存你输入的 Windows 路径
)

func SetWorkDir() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("❌ 获取当前工作目录失败:", err)
		return
	}
	WorkDir = wd

	M1Dir = filepath.Join(WorkDir, "M1")
	BuildDir = filepath.Join(M1Dir, "build")
	OutputDir = filepath.Join(M1Dir, "output")
	LDIDir = filepath.Join(OutputDir, "LDI")
	TxtDir = filepath.Join(OutputDir, "txt")

	removeIfExists(BuildDir)
	removeIfExists(OutputDir)

	dirs := []string{M1Dir, BuildDir, LDIDir, TxtDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("❌ 创建目录失败 [%s]: %v\n", d, err)
			return
		}
	}

	fmt.Println("✅ M1 工作空间初始化成功")
	// fmt.Println("    WorkDir  :", WorkDir)
	// fmt.Println("    M1Dir    :", M1Dir)
	// fmt.Println("    BuildDir :", BuildDir)
	// fmt.Println("    OutputDir:", OutputDir)
	// fmt.Println("    LDIDir   :", LDIDir)
	// fmt.Println("    TxtDir   :", TxtDir)
}

func removeIfExists(path string) {
	if _, err := os.Stat(path); err == nil {
		_ = os.RemoveAll(path)
	}
}
