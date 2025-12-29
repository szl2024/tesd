package M1_Public_Data

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	WorkDir   string	//代码运行位置，不是真实的M1的位置
	M1Dir     string	//M1的位置
	BuildDir  string	//M1下的Build文件夹的位置
	OutputDir string	//M1下的output文件夹的位置
	LDIDir    string	//M1的output文件夹下LDI文件夹的位置
	TxtDir    string	//M1的output文件夹下txt文件夹的位置

	SrcPath   string	//这里用来保存你输入的 Windows 路径，模型的路径
)

//设置工作空间
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
	
	//清除上一个项目留下的文件
	removeIfExists(BuildDir)
	removeIfExists(OutputDir)

	//创建新的文件夹
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
//如果路径下存在这个文件夹就清除
func removeIfExists(path string) {
	if _, err := os.Stat(path); err == nil {
		_ = os.RemoveAll(path)
	}
}
