package SWC_Dependence

import (
	"fmt"
	"strings"
	"FCU_Tools/LDI_Create"
	"github.com/xuri/excelize/v2"
)

type DependencyInfo struct {
	To            string
	Count         int
	InterfaceType string
}

//  M3/M6 사용: 각 연결은 독립적으로 유지되며, Count는 고정값 1이다.  

func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Excel 파일 열기 실패: %v", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("행 읽기 실패: %v", err)
	}

	type portInfo struct {
		component     string
		portType      string
		interfaceType string
	}
	deMap := make(map[string][]portInfo)
	for i, row := range rows {
		if i == 0 || len(row) < 12 {
			continue
		}
		component := strings.TrimSpace(row[3])
		portType := strings.TrimSpace(row[6])
		interfaceType := strings.TrimSpace(row[8])
		deOp := strings.TrimSpace(row[11])

		if component == "" || portType == "" || deOp == "" {
			continue
		}

		deMap[deOp] = append(deMap[deOp], portInfo{component, portType, interfaceType})
	}

	result := make(map[string][]DependencyInfo)

	for _, ports := range deMap {
		var pComp, rComp, interfaceType string
		for _, p := range ports {
			if p.portType == "P" {
				pComp = p.component
				interfaceType = p.interfaceType
			} else if p.portType == "R" {
				rComp = p.component
			}
		}
		if pComp != "" && rComp != "" && pComp != rComp {
			result[pComp] = append(result[pComp], DependencyInfo{
				To:            rComp,
				Count:         1,
				InterfaceType: interfaceType,
			})
		}
	}

	return result, nil
}

// ExtractDependenciesAggregatedFromASW 는 ASW Excel(filePath)을 읽어 포트 연결 정보를 파싱하고,
// 동일한 컴포넌트 쌍 사이의 여러 연결을 집계하여 “컴포넌트 → 컴포넌트” 의 의존성 목록을 생성한다.
//
// 처리 과정:
//   1) Excel 파일을 열고 첫 번째 시트의 모든 행을 읽는다.
//   2) 각 행에서 component / portType(P/R) / interfaceType / deOp(연결 식별자)를 추출하여,
//      임시 테이블 deMap[deOp] = []portInfo 형태로 저장한다.
//   3) deOp 단위로 그룹화하여 제공자(P) 컴포넌트와 수요자(R) 컴포넌트를 찾는다.
//      pComp 와 rComp 가 모두 존재하고 서로 다르면 의존성을 카운트한다:
//        - 이미 동일한 from→to 관계가 있으면 Count++
//        - 없으면 새로운 DependencyInfo{To, Count=1, InterfaceType}를 생성한다.
//   4) 결과를 map[from][]DependencyInfo 형태로 변환하여 반환한다.

func ExtractDependenciesAggregatedFromASW(filePath string) (map[string][]DependencyInfo, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Excel 파일 열기 실패: %v", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("행 읽기 실패: %v", err)
	}

	type portInfo struct {
		component     string
		portType      string
		interfaceType string
	}
	deMap := make(map[string][]portInfo)
	for i, row := range rows {
		if i == 0 || len(row) < 12 {
			continue
		}
		component := strings.TrimSpace(row[3])
		portType := strings.TrimSpace(row[6])
		interfaceType := strings.TrimSpace(row[8])
		deOp := strings.TrimSpace(row[11])

		if component == "" || portType == "" || deOp == "" {
			continue
		}

		deMap[deOp] = append(deMap[deOp], portInfo{component, portType, interfaceType})
	}

	countMap := make(map[string]map[string]*DependencyInfo)

	for _, ports := range deMap {
		var pComp, rComp, interfaceType string
		for _, p := range ports {
			if p.portType == "P" {
				pComp = p.component
				interfaceType = p.interfaceType
			} else if p.portType == "R" {
				rComp = p.component
			}
		}
		if pComp != "" && rComp != "" && pComp != rComp {
			if _, ok := countMap[pComp]; !ok {
				countMap[pComp] = make(map[string]*DependencyInfo)
			}
			if existing, ok := countMap[pComp][rComp]; ok {
				existing.Count++
			} else {
				countMap[pComp][rComp] = &DependencyInfo{
					To:            rComp,
					Count:         1,
					InterfaceType: interfaceType,
				}
			}
		}
	}

	result := make(map[string][]DependencyInfo)
	for from, depMap := range countMap {
		for _, dep := range depMap {
			result[from] = append(result[from], *dep)
		}
	}

	return result, nil
}

/* AnalyzeSWCDependencies ASW 엑셀(filePath)을 읽어 "컴포넌트 → 컴포넌트" 의존성을 추출합니다.
결과를 LDI에 필요한 두 매핑(depMap, strengthMap)으로 변환하고, 이를 기반으로 LDI XML 파일을 생성합니다.

// 프로세스:
1) ExtractDependenciesAggregatedFromASW를 호출하여 Excel의 포트 연결을 분석/집계합니다.
map[from][]DependencyInfo(각 DependencyInfo에는 대상 컴포넌트 To, 연결 횟수 Count, 인터페이스 유형 InterfaceType이 포함됨)를 얻습니다.
2) 집계 결과를 순회하며 구성:
- depMap       : map[string][]string           // from → 의존하는 목표 컴포넌트 목록
- strengthMap : map[string]map[string]int     // from → (to → 의존 강도/횟수)
3) LDI_Create.GenerateLDIXml(depMap, strengthMap)를 호출하여 LDI XML을 생성합니다(출력 파일명/경로는 해당 함수에 의해 결정됨).
*/
func AnalyzeSWCDependencies(filePath string) error {
	dependencies, err := ExtractDependenciesAggregatedFromASW(filePath)
	if err != nil {
		return err
	}

	// // 打印调试信息
	// for from, deps := range dependencies {
	// 	for _, dep := range deps {
	// 		fmt.Printf("%s -> %s (%s) x %d\n", from, dep.To, dep.InterfaceType, dep.Count)
	// 	}
	// }

	// LDI에 필요한 형식으로 변환  
	depMap := make(map[string][]string)
	strengthMap := make(map[string]map[string]int)

	for from, deps := range dependencies {
		for _, dep := range deps {
			depMap[from] = append(depMap[from], dep.To)
			if strengthMap[from] == nil {
				strengthMap[from] = make(map[string]int)
			}
			strengthMap[from][dep.To] = dep.Count
		}
	}

	// ldi.xml생성
	err = LDI_Create.GenerateLDIXml(depMap, strengthMap)
	if err != nil {
		return fmt.Errorf("LDI 파일 생성 실패: %v", err)
	}

	fmt.Println("✅ LDI 파일 생성 완료.")
	return nil
}

