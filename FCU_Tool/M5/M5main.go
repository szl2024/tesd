package M5main

import (
	"fmt"
	"FCU_Tools/M5/File_Utils_M5"
	"FCU_Tools/M5/LDI_M5_Create"
)

// M5_main 은 M5 지표의 총 진입점이다: 출력 디렉터리를 준비하고,  
// M5 LDI 파일을 생성한 후 주 LDI에 병합한다.  

func M5_main() {

	//   1) File_Utils_M5.PrepareM5OutputDir를 호출하여 출력 디렉토리를 초기화한다.  
	if err := File_Utils_M5.PrepareM5OutputDir(); err != nil {
		fmt.Println("5 출력 디렉토리 준비 실패: ", err)
		return
	}

	//   2) File_Utils_M5.GenerateM5LDIXml을 호출하여 component_info.csv을 읽고 M5.ldi.xml을 생성한다.  
	File_Utils_M5.GenerateM5LDIXml()

	//   3) LDI_M5_Create.MergeM5ToMainLDI를 호출하여 m5 및 m5demo 지표를 주 LDI 파일에 병합한다.  
	LDI_M5_Create.MergeM5ToMainLDI()
}
