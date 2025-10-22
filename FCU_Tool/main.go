package main

import (
	// "fmt"
	// "path/filepath"
	// "FCU_Tools/Public_data"
	// "FCU_Tools/SWC_Dependence"
	// "FCU_Tools/M2"
	// "FCU_Tools/M3"
	// "FCU_Tools/M4"
	// "FCU_Tools/M5"
	// "FCU_Tools/M6"
	"FCU_Tools/M1"
)

func main() {
	// /***************SWC 의존 관계***************/
	// // 분석 결과는 Main 프로잭트 디렉토리의 Output폴더에 생성함. Output풀더를 초기화(이미 있으면 삭제, 없으면 생성)
	// if err := Public_data.InitOutputDirectory(); err != nil {
	// 	fmt.Println("출력 디렉토리 초기화 실패: ", err)
	// 	return
	// }

	// // asw.xlsw는 각 컴포넌트의 연결 정보를 저장하니까 asw.xlsw를 저장하는 디렉토리를 입력함.
	// var dir string
	// fmt.Print("asw.xlsx를 저장할 폴더 경로를 입력하십시오: ")
	// fmt.Scanln(&dir)

	// // asw.xlsw의 경로를 Public_data.go 파일에 저장합니다.
	// excelPath := filepath.Join(dir, "asw.xlsx")
	// Public_data.SetConnectorFilePath(excelPath)

	// // asw.xlsw 파일 내용에 따라 각 컴포넌트 간의 의존 관계를 분석합니다. 구체적으로 컴포넌트 간 의존 강도 분석(ldi.xml에서 <uses provider="CL1MGR" strength="1"/>의 strength 값)
	// err := SWC_Dependence.AnalyzeSWCDependencies(excelPath)
	// if err != nil {
	// 	fmt.Println("의존관계 분석 실패: ", err)
	// } else {
	// 	fmt.Println("의존관계 분석 완료.")
	// }
	
	// /*
	// * 다음은 6가지 지표를 분석하는 코드의 호출 함수입니다.
	// * M1은 저장 모델의 경로를 기록합니다.
 	// * M2는 complexity.json 및 rq_versus_component.xlsx의 경로를 기록합니다.
 	// * M3는 component_info.xlsx의 경로를 기록합니다.
 	// * M4-M6은 어떤 경로도 기록하지 않고 M2-M3에 입력된 경로를 직접 사용합니다. 
 	//   M2 또는 M3을 사용하지 않으면 관련 함수를 조정하여 파일 경로를 함수에 전달해야 합니다.
	// */

	/***************M1지표***************/
	M1main.M1_main()
	// /***************M2지표***************/
	// M2main.M2_main()
	// /***************M3지표***************/
	// M3main.M3_main()
	// /***************M4지표***************/
	// M4main.M4_main()
	// /***************M5지표***************/
	// M5main.M5_main()
	// /***************M6지표***************/
	// M6main.M6_main()
}