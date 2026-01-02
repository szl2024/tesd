package M6main

import (
	"fmt"
	"FCU_Tools/M6/File_Utils_M6"
	"FCU_Tools/M6/LDI_M6_Create"
)
// M6_main 은 M6 프로세스의 총 진입점이다: 출력 디렉터리를 준비하고,  
// M6 LDI 파일을 생성한 후 주 LDI에 병합한다.  

func M6_main() {
	//   1) File_Utils_M6.PrepareM6OutputDir를 호출하여 출력 디렉터리를 초기화한다.  
	if err := File_Utils_M6.PrepareM6OutputDir(); err != nil {
		fmt.Println("6 출력 디렉토리 준비 실패: ", err)
		return
	}

	//   2) File_Utils_M6.GenerateM6LDIXml을 호출하여 M6 지표를 계산하고 M6.ldi.xml 및 M6.txt를 생성한다.  
	File_Utils_M6.GenerateM6LDIXml()
	
	//   3) LDI_M6_Create.MergeM6ToMainLDI를 호출하여 M6 지표를 주 LDI 파일 result.ldi.xml에 병합한다.  
	LDI_M6_Create.MergeM6ToMainLDI()


}