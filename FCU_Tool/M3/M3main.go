package M3main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"FCU_Tools/M3/File_Utils_M3"
	"FCU_Tools/M3/LDI_M3_Create"
)
// M3_main 은 M3 프로세스의 진입점이다: 입력을 검사하고 출력 디렉터리를 준비하며,  
// M3 LDI 파일을 생성하고 그 지표를 주 LDI에 병합한다.  


func M3_main() {
	//   1) 사용자에게 component_info.xlsx가 포함된 디렉터리를 입력하도록 안내한다.    
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("필요한 M3 파일(component_info.xlsx)이 포함된 폴더 경로를 입력하십시오: ")
	dirInput, _ := reader.ReadString('\n')
	dir := strings.TrimSpace(dirInput)

	//   2) File_Utils_M3.CheckAndSetM2InputPath를 호출하여 입력 파일 경로를 검증하고 저장한다.  
	if err := File_Utils_M3.CheckAndSetM2InputPath(dir); err != nil {
		fmt.Println("M3 가져오기 파일 설정 실패: ", err)
		return
	}

	//   3) File_Utils_M3.PrepareM2OutputDir를 호출하여 출력 디렉터리를 삭제하고 다시 생성한다.   
	if err := File_Utils_M3.PrepareM2OutputDir(); err != nil {
		fmt.Println("M3 출력 디렉토리 준비 실패：", err)
		return
	}

	//   4) File_Utils_M3.GenerateM3LDIXml을 호출하여 M3/output/M3.ldi.xml과 M3.txt를 생성한다.  
	File_Utils_M3.GenerateM3LDIXml()

	//   5) LDI_M3_Create.MergeM3ToMainLDI를 호출하여 M3 지표를 주 LDI 파일에 병합한다.   
	LDI_M3_Create.MergeM3ToMainLDI()


}