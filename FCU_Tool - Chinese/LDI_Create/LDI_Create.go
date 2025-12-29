package LDI_Create

import (
	"FCU_Tools/Public_data"
	"fmt"
	"os"
	"path/filepath"
)


func GenerateLDIXml(dependencies map[string][]string, strengths map[string]map[string]int) error {
	//将输出的内容，即ldi.xml文件的正确路径进行拼接
	outputPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")
	//将outputPath中存储的ldi.xml文件进行生成
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer file.Close()
	
	//ldi.xml的固定格式
	_, _ = file.WriteString("<ldi>\n")

	//将内容进行写入，根据两个map
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
	//ldi.xml的固定格式
	_, _ = file.WriteString("</ldi>\n")

	fmt.Println("LDI 파일이 기록됨：", outputPath)
	return nil
}