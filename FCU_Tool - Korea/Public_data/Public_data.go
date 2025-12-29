// Public_data.go
package Public_data

import (
	"fmt"
	"os"
	"path/filepath"
)

var HierarchyTable [][]string

// ConnectorFilePath에는 asw.csv의 경로가 기록되어 있습니다.
var ConnectorFilePath string

// OutputDir에는 최중 출력 경로가 기록되어 있습니다.
var OutputDir string

// ComplexityJsonPath에는 complexity.json의 경로가 기록되어 있습니다.
var M2ComplexityJsonPath string

// RqExcelPath에는 rq_versus_component.xlsx의 경로가 기록되어 있습니다.
var M2RqExcelPath string

// M3component_infoxlsxPath에는 component_info.xlsx의 경로가 기록되어 있습니다.
var M3component_infoxlsxPath string

//여기에는 M2~M6 각각의 출력 경로가 기록되어 있습니다.
var M2OutputlPath string
var M3OutputlPath string
var M4OutputlPath string
var M5OutputlPath string
var M6OutputlPath string

// SetConnectorFilePath를 통해 asw.csv 경로를 설정합니다.
func SetConnectorFilePath(path string) {
	ConnectorFilePath = path

}
func SetM2M3FilePath(path string) {
	M2ComplexityJsonPath = filepath.Join(path, "complexity.json")
	M2RqExcelPath = filepath.Join(path, "rq_versus_component.csv")
	M3component_infoxlsxPath = filepath.Join(path, "component_info.csv")
}
// 터미널에 asw.csv 파일의 경로를 입력하고, 해당 경로를 ConnectorFilePath에 기록합니다.
func InitConnectorFilePathFromUser() error {
	var dir string
	fmt.Print("asw.csv 등 파일의 경로를 입력하세요:")
	if _, err := fmt.Scanln(&dir); err != nil {
		return fmt.Errorf("읽기 실패: %v", err)
	}

	csvPath := filepath.Join(dir, "asw.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("asw.csv 파일을 찾을 수 없습니다: %s", csvPath)
	}

	SetConnectorFilePath(csvPath)
	SetM2M3FilePath(dir)
	return nil
}

// // SetM2InputDir는 complexity.json 및 rq_versus_component.xlsx가 포함된 사용자 입력의 디렉토리 경로를 설정합니다.
// func SetM2InputDir(dir string) error {
// 	complexity := filepath.Join(dir, "complexity.json")
// 	rqExcel := filepath.Join(dir, "rq_versus_component.xlsx")

// 	if _, err := os.Stat(complexity); os.IsNotExist(err) {
// 		return fmt.Errorf("complexity.json을 찾을 수 없습니다: %s", complexity)
// 	}
// 	if _, err := os.Stat(rqExcel); os.IsNotExist(err) {
// 		return fmt.Errorf("rq_versus_component.xlsx 을 찾을 수 없습니다.: %s", rqExcel)
// 	}

// 	M2ComplexityJsonPath = complexity
// 	M2RqExcelPath = rqExcel
// 	return nil
// }

// Output 경로를 초기화하고, asw.csv 경로를 기록하는 함수 호출
func InitOutputDirectory() {
	//루트 디렉터리를 받음
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Println("출력 디렉터리 초기화에 실패했습니다: 현재 작업 디렉터리를 가져올 수 없습니다.", err)
		return
	}
	//루트 디렉터리 + Output를 통해 출력경로를 만듬.
	outputPath := filepath.Join(baseDir, "Output")
	OutputDir = outputPath

	// Output 디렉터리가 존재하면 삭제합니다
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			fmt.Println("출력 디렉터리 초기화에 실패했습니다: 이전 Output 디렉터리 삭제에 실패했습니다:", err)
			return
		}
	}

	// Output 디렉터리 생성
	if err := os.Mkdir(outputPath, 0755); err != nil {
		fmt.Println("출력 디렉터리 초기화에 실패했습니다: Output 디렉터리 생성에 실패했습니다:", err)
		return
	}

	// asw.csv 경로 입력 및 기록
	if err := InitConnectorFilePathFromUser(); err != nil {
		fmt.Println("asw.csv 경로 설정에 실패했습니다:", err)
		return
	}
}
