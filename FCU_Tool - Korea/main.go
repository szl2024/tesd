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
	/***************SWC간 의존관계***************/
	// 1) Output 폴더를 초기화 (여기에서도 asw.csv가 어디는지를 물어보고 경로를 Public_data.ConnectorFilePath에 저장함)
	Public_data.InitOutputDirectory()

	// 2) Public_data.ConnectorFilePath에 저장된 asw.csv를 기반으로 swc간 의존관계를 분석함.
	SWC_Dependence.AnalyzeSWCDependencies(Public_data.ConnectorFilePath)

	/***************M1지표***************/
	M1main.M1_main()

	/***************M2지표***************/
	M2main.M2_main()

	/***************M3지표***************/
	M3main.M3_main()

	/***************M4지표***************/
	M4main.M4_main()

	/***************M5지표***************/
	M5main.M5_main()

	/***************M6지표***************/
	M6main.M6_main()
}
