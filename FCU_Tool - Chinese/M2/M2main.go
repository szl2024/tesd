package M2main

import (
	"fmt"
	"path/filepath"
	"strings"

	"FCU_Tools/M2/File_Utils_M2"
	"FCU_Tools/M2/LDI_M2_Create"
	"FCU_Tools/Public_data"
)

func M2_main() {
	// 从 Public_data.ConnectorFilePath（例如 D:\test\testset\asw.csv）推导输入目录
	base := strings.TrimSpace(Public_data.ConnectorFilePath)
	if base == "" {
		fmt.Println("M2 자동 경로 설정 실패: ConnectorFilePath가 비어 있습니다.")
		return
	}
	dir := filepath.Dir(base)

	// 继续沿用原逻辑：检查并设置 M2 输入路径（complexity.json / rq_versus_component.csv）
	if err := File_Utils_M2.CheckAndSetM2InputPath(dir); err != nil {
		fmt.Println("M2 가져오기 파일 설정 실패: ", err)
		return
	}

	if err := File_Utils_M2.PrepareM2OutputDir(); err != nil {
		fmt.Println("M2 출력 디렉토리 준비 실패：", err)
		return
	}

	File_Utils_M2.GenerateM2LDIXml()
	LDI_M2_Create.MergeM2ToMainLDI()
}
