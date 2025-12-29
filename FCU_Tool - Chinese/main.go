// main.go
package main

import (
	"FCU_Tools/M1"
	"FCU_Tools/M2"
	"FCU_Tools/M3"
	"FCU_Tools/M4"
	"FCU_Tools/M5"
	"FCU_Tools/M6"
	"FCU_Tools/Public_data"
	"FCU_Tools/SWC_Dependence"
)

func main() {
	/***************SWC 依赖关系***************/
	// 1) 初始化 Output（内部会询问 asw.csv 所在文件夹并设置 Public_data.ConnectorFilePath）
	Public_data.InitOutputDirectory()

	// 2) 直接用 Public_data.ConnectorFilePath 做依赖分析（成功/失败打印在函数内部）
	SWC_Dependence.AnalyzeSWCDependencies(Public_data.ConnectorFilePath)

	/***************M1指标***************/
	M1main.M1_main()
	/***************M2指标***************/
	M2main.M2_main()
	/***************M3指标***************/
	M3main.M3_main()
	/***************M4指标***************/
	M4main.M4_main()
	/***************M5指标***************/
	M5main.M5_main()
	/***************M6指标***************/
	M6main.M6_main()
}
