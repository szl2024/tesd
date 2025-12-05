package Public_data

import (
	"fmt"
	"os"
	"path/filepath"
)
var HierarchyTable [][]string

// ConnectorFilePath는 사용자가 입력한 asw.csv 파일 경로를 저장합니다.
var ConnectorFilePath string

// OutputDir는 프로그램이 자동으로 만드는 출력 디렉토리 경로입니다.
var OutputDir string

// ComplexityJsonPath는 complexity.json의 경로입니다.
var M2ComplexityJsonPath string

// RqExcelPath는 rq_versus_component.xlsx의 경로입니다.
var M2RqExcelPath string

var M3component_infoxlsxPath string

var M2OutputlPath string
var M3OutputlPath string
var M4OutputlPath string
var M5OutputlPath string
var M6OutputlPath string



// SetConnectorFilePath 사용자가 입력한 connector.xlsx 파일 경로 설정
func SetConnectorFilePath(path string) {
	ConnectorFilePath = path
}

// SetM2InputDir는 complexity.json 및 rq_versus_component.xlsx가 포함된 사용자 입력의 디렉토리 경로를 설정합니다.
func SetM2InputDir(dir string) error {
	complexity := filepath.Join(dir, "complexity.json")
	rqExcel := filepath.Join(dir, "rq_versus_component.xlsx")

	if _, err := os.Stat(complexity); os.IsNotExist(err) {
		return fmt.Errorf("complexity.json을 찾을 수 없습니다: %s", complexity)
	}
	if _, err := os.Stat(rqExcel); os.IsNotExist(err) {
		return fmt.Errorf("rq_versus_component.xlsx 을 찾을 수 없습니다.: %s", rqExcel)
	}

	M2ComplexityJsonPath = complexity
	M2RqExcelPath = rqExcel
	return nil
}

// InitOutputDirectory 는 출력 디렉토리 Output 을 초기화하고 존재하는 경우 재구성 비우기
func InitOutputDirectory() error {
	// 현재 작업 디렉토리 가져오기 (프로젝트 루트)
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("현재 작업 디렉토리를 가져오지 못했습니다.: %v", err)
	}

	outputPath := filepath.Join(baseDir, "Output")
	OutputDir = outputPath

	// Output 디렉토리가 이미 있으면 삭제합니다.
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("이전 Output 디렉토리를 삭제하지 못했습니다.: %v", err)
		}
	}

	// 새 Output 디렉토리 만들기
	if err := os.Mkdir(outputPath, 0755); err != nil {
		return fmt.Errorf("새 Output 디렉토리를 생성하지 못했습니다.: %v", err)
	}

	return nil
}
