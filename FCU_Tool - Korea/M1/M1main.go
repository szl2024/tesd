package M1main

import (
	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/File_Utils_M1"
	"FCU_Tools/M1/Analysis_Process"
	"FCU_Tools/M1/LDI_M1_Create"
)

func M1_main() {
	// 1. 작업 공간 생성: M1/Build, M1/Output/LDI, M1/Output/txt
	M1_Public_Data.SetWorkDir()

	// 2. 모델의 Windows 경로 읽기
	File_Utils_M1.ReadWindowsPath()
	
	// 3. 요구 조건을 만족하는 slx 파일을 BuildDir로 복사합니다.
	File_Utils_M1.CopySlxToBuild()

	// 4. slx 파일을 BuildDir 아래의 동일한 이름의 디렉터리로 압축 해제합니다.
	File_Utils_M1.UnzipSlxFiles()

	// 5. 분석 흐름을 설정하며, 파라미터에 따라 분석 깊이가 결정됩니다. 
	// 다만 현재 요구사항이 3단계(3층)까지이므로, 테스트는 3단계까지만 수행했습니다.
	Analysis_Process.RunAnalysis(3)

	// 6. txt 파일을 기반으로 ldi.xml 파일을 생성합니다.
	File_Utils_M1.GenerateM1LDIFromTxt()

	// 7. M1의 ldi.xml을 주(메인) ldi.xml에 병합합니다.
	LDI_M1_Create.MergeM1ToMainLDI()
}
