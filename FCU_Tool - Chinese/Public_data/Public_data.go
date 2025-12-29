// Public_data.go
package Public_data

import (
	"fmt"
	"os"
	"path/filepath"
)

var HierarchyTable [][]string

// ConnectorFilePath记录了asw.csv的路径
var ConnectorFilePath string

// OutputDir记录了主输出的路径
var OutputDir string

// ComplexityJsonPath记录了complexity.json的路径
var M2ComplexityJsonPath string

// RqExcelPath记录了rq_versus_component.xlsx的路径
var M2RqExcelPath string

// M3component_infoxlsxPath记录了component_info.xlsx的路径
var M3component_infoxlsxPath string

//这里记录了M2 - M6的各自输出的路径
var M2OutputlPath string
var M3OutputlPath string
var M4OutputlPath string
var M5OutputlPath string
var M6OutputlPath string

// 通过SetConnectorFilePath来设置asw.csv的路径
func SetConnectorFilePath(path string) {
	ConnectorFilePath = path

}
func SetM2M3FilePath(path string) {
	M2ComplexityJsonPath = filepath.Join(path, "complexity.json")
	M2RqExcelPath = filepath.Join(path, "rq_versus_component.csv")
	M3component_infoxlsxPath = filepath.Join(path, "component_info.csv")
}
// 在控制台输入asw.csv文件的路径，并将其记录在ConnectorFilePath中。
func InitConnectorFilePathFromUser() error {
	var dir string
	fmt.Print("请输入asw.csv文件的路径: ")
	if _, err := fmt.Scanln(&dir); err != nil {
		return fmt.Errorf("读取失败: %v", err)
	}

	csvPath := filepath.Join(dir, "asw.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("找不到asw.csv文件: %s", csvPath)
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

// 初始化Output路径，并且将asw.csv的路径记录函数调用
func InitOutputDirectory() {
	//获取项目根目录
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Println("初始化输出目录失败：无法获取当前工作目录。", err)
		return
	}
	//将根目录 + Output创建输出路径
	outputPath := filepath.Join(baseDir, "Output")
	OutputDir = outputPath

	// Output目录存在的话，就删除
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.RemoveAll(outputPath); err != nil {
			fmt.Println("初始化输出目录失败：上一个 Output目录删除失败:", err)
			return
		}
	}

	// 新建Output目录
	if err := os.Mkdir(outputPath, 0755); err != nil {
		fmt.Println("初始化输出目录失败： 新建Output目录失败:", err)
		return
	}

	// asw.csv 输入并记录
	if err := InitConnectorFilePathFromUser(); err != nil {
		fmt.Println("asw.csv路径设置失败:", err)
		return
	}
}
