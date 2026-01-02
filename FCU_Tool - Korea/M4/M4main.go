package M4main

import (
	"fmt"
	"FCU_Tools/M4/File_Utils_M4"
	"FCU_Tools/M4/LDI_M4_Create"
)
// M4_main 은 M4 지표 계산과 병합의 총 진입점이다.  
func M4_main() {

	//   1) File_Utils_M4.PrepareM4OutputDir를 호출하여 출력 디렉터리를 초기화한다.  
 	if err := File_Utils_M4.PrepareM4OutputDir(); err != nil {
		fmt.Println("M4 출력 디렉토리 준비 실패：", err)
		return
	}

	//   2) File_Utils_M4.GenerateM4LDIXml을 호출하여 지표를 계산하고 M4.ldi.xml과 M4.txt를 생성한다.  
	File_Utils_M4.GenerateM4LDIXml()

	//   3) LDI_M4_Create.MergeM4ToMainLDI를 호출하여 결과를 주 LDI 파일 result.ldi.xml에 병합한다.  
	LDI_M4_Create.MergeM4ToMainLDI()


}