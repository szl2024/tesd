package LDI_Create

import (
	"FCU_Tools/Public_data"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"io/ioutil"
	"sort"
	"FCU_Tools/M1/M1_Public_Data"
)

// GenerateLDIXml 주어진 의존성 정보를 기반으로 LDI XML(result.ldi.xml)을 생성한다.
//
// 처리 과정:
//   1) Public_data.OutputDir 경로 아래에 result.ldi.xml 파일을 생성한다.
//   2) dependencies 맵을 순회하면서 각 user(사용자 컴포넌트)에 대해 <element name="..."> 블록을 작성한다.
//   3) 각 provider(제공자 컴포넌트)에 대해 <uses provider="..." strength="..."/> 태그를 출력한다.
//        - strength 값은 strengths[user][provider]에서 가져오며, 없을 경우 기본값 1을 기록한다.
//   4) 모든 element 블록을 닫고 </ldi> 루트 태그를 추가한 후 파일을 완성한다.
//
func GenerateLDIXml(dependencies map[string][]string, strengths map[string]map[string]int) error {
	outputPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer file.Close()

	_, _ = file.WriteString("<ldi>\n")

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

	_, _ = file.WriteString("</ldi>\n")

	fmt.Println("LDI 파일이 기록됨：", outputPath)
	return nil
}

// MergeAdditionalLDI 보충 LDI(additionPath)를 주 LDI(mainPath)에 병합하여
// 요소 중복을 제거하고 상위 모듈 단위로 정렬된 LDI XML을 재작성한다.
//
// 처리 과정:
//   1) mainPath, additionPath 두 LDI XML 파일 내용을 읽는다.
//   2) parseElements 내부 함수를 통해 <element name="..."> 블록을 파싱하여 elementMap[name]에 누적한다.
//   3) 모든 element 이름을 상위 모듈(예: "CL1CM1.CL1CLS1" → "CL1CM1") 기준으로 groupMap에 분류한다.
//   4) 모듈/요소 이름을 정렬한 뒤, 중복 없는 <uses .../> 태그만 남기고 다시 <ldi> 구조로 빌드한다.
//   5) 최종적으로 mainPath 파일을 새로운 병합 결과로 덮어쓴다.
//
func MergeAdditionalLDI(mainPath, additionPath string) error {
	mainData, err := ioutil.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("주 LDI 파일 읽기 실패: %v", err)
	}
	additionData, err := ioutil.ReadFile(additionPath)
	if err != nil {
		return fmt.Errorf("보충 LDI 파일 읽기 실패: %v", err)
	}

	elementMap := make(map[string][]string)

	parseElements := func(data string) {
		lines := strings.Split(data, "\n")
		var name string
		var content []string
		collect := false
		for _, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "<element name=") {
				if collect && name != "" {
					elementMap[name] = append(elementMap[name], content...)
				}
				name = getElementName(trim)
				content = []string{line}
				collect = true
			} else if collect {
				content = append(content, line)
				if trim == "</element>" {
					elementMap[name] = append(elementMap[name], content...)
					collect = false
					name = ""
					content = nil
				}
			}
		}
		if collect && name != "" {
			elementMap[name] = append(elementMap[name], content...)
		}
	}

	parseElements(string(mainData))
	parseElements(string(additionData))

	// 모든 요소를 상위 모듈로 분류합니다(예: CL1CM1.CL1CLS1은 CL1CM1).
	groupMap := make(map[string][]string)
	for name := range elementMap {
		parts := strings.Split(name, ".")
		group := parts[0] 
		groupMap[group] = append(groupMap[group], name)
	}

	var modules []string
	for mod := range groupMap {
		modules = append(modules, mod)
	}
	sort.Strings(modules)

	var builder strings.Builder
	builder.WriteString("<ldi>\n")
	for _, mod := range modules {
		sort.Strings(groupMap[mod])
		for _, name := range groupMap[mod] {
			builder.WriteString(fmt.Sprintf("  <element name=\"%s\">\n", name))
			seen := make(map[string]bool)
			for _, line := range elementMap[name] {
				trim := strings.TrimSpace(line)
				if trim == fmt.Sprintf("<element name=\"%s\">", name) || trim == "</element>" {
					continue
				}
				if !seen[trim] {
					builder.WriteString("    " + trim + "\n")
					seen[trim] = true
				}
			}
			builder.WriteString("  </element>\n")
		}
	}
	builder.WriteString("</ldi>\n")

	return os.WriteFile(mainPath, []byte(builder.String()), 0644)
}

// getElementName <element name="..."> 문자열에서 name 속성 값을 추출한다.
//
// 처리 과정:
//   1) 문자열 좌우 공백을 제거한다.
//   2) 첫 번째 따옴표(") 위치와 마지막 따옴표(") 위치를 찾는다.
//   3) 두 인덱스 사이의 부분 문자열을 반환한다.
//   4) 포맷이 올바르지 않으면 빈 문자열을 반환한다.
//
func getElementName(line string) string {
	line = strings.TrimSpace(line)
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start != -1 && end != -1 && end > start {
		return line[start+1 : end]
	}
	return ""
}

// MergeAllFromM1LDIFolder M1 LDI 출력 폴더의 모든 보충 LDI(.ldi.xml) 파일을 읽어
// 기본 LDI(result.ldi.xml)에 순차적으로 병합한다.
//
// 처리 과정:
//   1) Public_data.OutputDir/result.ldi.xml 을 마스터로 사용한다.
//   2) M1_Public_Data.OutputDir/LDI 디렉토리를 읽어 .ldi.xml 파일들을 찾는다.
//   3) 각 파일에 대해 MergeAdditionalLDI(mainPath, addPath)를 호출하여 순차 병합한다.
//   4) 모든 병합이 끝나면 콘솔에 완료 메시지를 출력한다.
//
func MergeAllFromM1LDIFolder() error {
	mainPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")
	tempMain := mainPath

	addDir := filepath.Join(M1_Public_Data.OutputDir, "LDI")
	files, err := ioutil.ReadDir(addDir)
	if err != nil {
		return fmt.Errorf("보충 LDI 디렉토리 읽기 실패: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".ldi.xml") {
			addPath := filepath.Join(addDir, file.Name())
			err = MergeAdditionalLDI(tempMain, addPath)
			if err != nil {
				return err
			}
		}
	}

	//fmt.Println("모두 병합 완료, 파일 내보내기: ", tempMain)
	return nil
}
