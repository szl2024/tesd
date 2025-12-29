package M3main

import (
	"fmt"
	//"path/filepath"
	"strings"

	"FCU_Tools/M3/File_Utils_M3"
	"FCU_Tools/M3/LDI_M3_Create"
	"FCU_Tools/Public_data"
)

func M3_main() {
	base := strings.TrimSpace(Public_data.ConnectorFilePath)
	if base == "" {
		fmt.Println("M3 자동 경로 설정 실패: ConnectorFilePath가 비어 있습니다.")
		return
	}
	// dir := filepath.Dir(base)

	// 기존 로직을 계속 사용합니다: M3 입력 경로(component_info.csv)를 확인하고 설정합니다.
	// if err := File_Utils_M3.CheckAndSetM3InputPath(dir); err != nil {
	// 	fmt.Println("M3 가져오기 파일 설정 실패: ", err)
	// 	return
	// }

	if err := File_Utils_M3.PrepareM3OutputDir(); err != nil {
		fmt.Println("M3 출력 디렉토리 준비 실패：", err)
		return
	}

	File_Utils_M3.GenerateM3LDIXml()
	LDI_M3_Create.MergeM3ToMainLDI()
}
