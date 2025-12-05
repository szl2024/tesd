package M1main

import (
	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/File_Utils_M1"
	"FCU_Tools/M1/Analysis_Process"
	"FCU_Tools/M1/LDI_M1_Create"
)

func M1_main() {
	// 1. 创建工作空间：M1/Build、M1/Output/LDI、M1/Output/txt
	M1_Public_Data.SetWorkDir()

	// 2. 读取 Windows 路径
	File_Utils_M1.ReadWindowsPath()
	
	// 3. 复制符合要求的 slx 文件到 BuildDir
	File_Utils_M1.CopySlxToBuild()

	// 4. 解压 slx 文件到 BuildDir 下同名目录
	File_Utils_M1.UnzipSlxFiles()

	// 5. 分析流程设定，参数决定分析的深度，但是只测试到第三层，因为目前的需求是前三层的内容
	Analysis_Process.RunAnalysis(3)

	// 6. 根据txt文件生成ldi.xml文件
	File_Utils_M1.GenerateM1LDIFromTxt()

	// 7. 将M1的ldi.xml合并到主ldi.xml
	LDI_M1_Create.MergeM1ToMainLDI()
}
