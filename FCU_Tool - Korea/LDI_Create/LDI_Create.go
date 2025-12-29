package LDI_Create

import (
	"FCU_Tools/Public_data"
	"fmt"
	"os"
	"path/filepath"
)


func GenerateLDIXml(dependencies map[string][]string, strengths map[string]map[string]int) error {
	//출력 결과인 ldi.xml 파일의 올바른 경로를 조합(결합)합니다.
	outputPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")
	//outputPath에 저장된 경로에 ldi.xml 파일을 생성합니다.
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer file.Close()
	
	//ldi.xml의 고정 형식
	_, _ = file.WriteString("<ldi>\n")

	//두 개의 map을 기반으로 내용을 작성(기록)합니다.
	for user, providers := range dependencies {
		_, _ = file.WriteString(fmt.Sprintf("  <element name=\"%s\">\n", user))
		for _, provider := range providers {
			if strengthVal, ok := strengths[user][provider]; ok {
    			// 항상 strength를 쓰다.
    			_, _ = file.WriteString(fmt.Sprintf("    <uses provider=\"%s\" strength=\"%d\"/>\n", provider, strengthVal))
			} else {
    			// 강도를 찾지 못하면 기본값으로 1을 작성합니다.
    			_, _ = file.WriteString(fmt.Sprintf("    <uses provider=\"%s\" strength=\"1\"/>\n", provider))
			}
		}
		_, _ = file.WriteString("  </element>\n")
	}
	//ldi.xml의 고정 형식
	_, _ = file.WriteString("</ldi>\n")

	fmt.Println("LDI 파일이 기록됨：", outputPath)
	return nil
}