package M2main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"FCU_Tools/M2/File_Utils_M2"
	"FCU_Tools/M2/LDI_M2_Create"
)
// M2_main은 M2 프로세스의 총 진입점으로,
// 사용자 입력 경로를 읽고, M2 입력 파일을 확인하며, 출력 디렉터리를 준비하고,
// M2 LDI 파일을 생성한 뒤 그 지표를 메인 LDI에 병합한다.


func M2_main() {
	//   1) 표준 입력에서 사용자가 지정한 디렉터리 경로를 읽는다
	//      (complexity.json과 rq_versus_component.xlsx를 포함해야 함).

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("필요한 M2 파일(complexity.json 및 rq_versus_component.xlsx)이 포함된 폴더 경로를 입력하십시오: ")
	dirInput, _ := reader.ReadString('\n')
	dir := strings.TrimSpace(dirInput)

	//   2) File_Utils_M2.CheckAndSetM2InputPath를 호출하여
	//      디렉터리를 검증하고 입력 파일 경로를 저장한다.

	if err := File_Utils_M2.CheckAndSetM2InputPath(dir); err != nil {
		fmt.Println("M2 가져오기 파일 설정 실패: ", err)
		return
	}

	//   3) File_Utils_M2.PrepareM2OutputDir를 호출하여
	//      출력 디렉터리를 삭제하고 다시 생성한다.

	if err := File_Utils_M2.PrepareM2OutputDir(); err != nil {
		fmt.Println("M2 출력 디렉토리 준비 실패：", err)
		return
	}

	//   4) File_Utils_M2.GenerateM2LDIXml을 호출하여
	//      M2/output/M2.ldi.xml을 생성한다.

	File_Utils_M2.GenerateM2LDIXml()

	//   5) LDI_M2_Create.MergeM2ToMainLDI를 호출하여
	//      coverage.m2를 메인 LDI에 병합한다.

	LDI_M2_Create.MergeM2ToMainLDI()


}