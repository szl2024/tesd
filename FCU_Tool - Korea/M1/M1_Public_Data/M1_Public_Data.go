package M1_Public_Data

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	WorkDir   string	//코드가 실행되는 위치로, 실제 M1의 위치가 아닙니다.
	M1Dir     string	//M1의 위치
	BuildDir  string	//M1 하위의 Build 폴더 위치
	OutputDir string	//M1 폴더 내 output 폴더의 위치
	LDIDir    string	//M1의 output 폴더 내 LDI 폴더 위치
	TxtDir    string	//M1의 output 폴더 내 txt 폴더 위치

	SrcPath   string	//여기에는 사용자가 입력한 Windows 경로(모델 경로)를 저장합니다.
)

//작업 공간 설정
func SetWorkDir() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("❌ 현재 작업 디렉터리를 가져오는 데 실패했습니다.", err)
		return
	}
	WorkDir = wd

	M1Dir = filepath.Join(WorkDir, "M1")
	BuildDir = filepath.Join(M1Dir, "build")
	OutputDir = filepath.Join(M1Dir, "output")
	LDIDir = filepath.Join(OutputDir, "LDI")
	TxtDir = filepath.Join(OutputDir, "txt")
	
	//이전 프로젝트에서 남아 있는 파일을 삭제합니다.
	removeIfExists(BuildDir)
	removeIfExists(OutputDir)

	//새 폴더를 생성합니다.
	dirs := []string{M1Dir, BuildDir, LDIDir, TxtDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("❌ 디렉터리 생성 실패 [%s]: %v\n", d, err)
			return
		}
	}

	//fmt.Println("✅ M1 작업 공간 초기화 성공")
	// fmt.Println("    WorkDir  :", WorkDir)
	// fmt.Println("    M1Dir    :", M1Dir)
	// fmt.Println("    BuildDir :", BuildDir)
	// fmt.Println("    OutputDir:", OutputDir)
	// fmt.Println("    LDIDir   :", LDIDir)
	// fmt.Println("    TxtDir   :", TxtDir)
}
//해당 경로에 이 폴더가 존재하면 삭제(정리)합니다.
func removeIfExists(path string) {
	if _, err := os.Stat(path); err == nil {
		_ = os.RemoveAll(path)
	}
}
