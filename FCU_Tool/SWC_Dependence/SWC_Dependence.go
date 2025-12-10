package SWC_Dependence

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"FCU_Tools/LDI_Create"
)

type DependencyInfo struct {
	To            string
	Count         int
	InterfaceType string
}

// ASW CSV 파일을 읽어 [][]string 형태로 반환
func loadASWRowsFromCSV(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("CSV 파일 열기 실패: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// 각 행마다 컬럼 수가 달라도 읽을 수 있도록 설정
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 행 읽기 실패: %v", err)
	}
	return rows, nil
}

//  M3/M6 사용: 각 연결은 독립적으로 유지되며, Count는 고정값 1이다.
func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component     string
		portType      string
		interfaceType string
	}

	deMap := make(map[string][]portInfo)
	for i, row := range rows {
		// i == 0 : 헤더, len(row) < 12 : 필요한 컬럼이 부족한 행은 스킵
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

		deMap[deOp] = append(deMap[deOp], portInfo{
			component:     component,
			portType:      portType,
			interfaceType: interfaceType,
		})
	}

	result := make(map[string][]DependencyInfo)

	// deOp 단위로 1→N 또는 N→1 관계 처리
	for _, ports := range deMap {
		var providers []portInfo
		var receivers []portInfo

		// P / R 분류
		for _, p := range ports {
			switch p.portType {
			case "P":
				providers = append(providers, p)
			case "R":
				receivers = append(receivers, p)
			}
		}

		// P 또는 R이 하나도 없으면 스킵
		if len(providers) == 0 || len(receivers) == 0 {
			continue
		}

		switch {
		// 1 P, N R (1→N)
		case len(providers) == 1 && len(receivers) >= 1:
			p := providers[0]
			for _, r := range receivers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					// 자기 자신 의존은 스킵
					continue
				}
				result[p.component] = append(result[p.component], DependencyInfo{
					To:            r.component,
					Count:         1,
					InterfaceType: p.interfaceType, // P 쪽 인터페이스 타입 사용
				})
			}

		// N P, 1 R (N→1)
		case len(receivers) == 1 && len(providers) >= 1:
			r := receivers[0]
			for _, p := range providers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					continue
				}
				result[p.component] = append(result[p.component], DependencyInfo{
					To:            r.component,
					Count:         1,
					InterfaceType: p.interfaceType,
				})
			}

		// 이론상 N P, M R (N>1,M>1)는 없다고 가정하지만, 방어적으로 스킵
		default:
			continue
		}
	}

	return result, nil
}

// ExtractDependenciesAggregatedFromASW 는 ASW CSV(filePath)을 읽어 포트 연결 정보를 파싱하고,
// 동일한 컴포넌트 쌍 사이의 여러 연결을 집계하여 “컴포넌트 → 컴포넌트” 의 의존성 목록을 생성한다.
//
// 처리 과정:
//   1) CSV 파일을 읽어 모든 행을 가져온다.
//   2) 각 행에서 component / portType(P/R) / interfaceType / deOp(연결 식별자)를 추출하여,
//      임시 테이블 deMap[deOp] = []portInfo 형태로 저장한다.
//   3) deOp 단위로 그룹화하여 제공자(P) 컴포넌트와 수요자(R) 컴포넌트를 찾는다.
//      (1P 다수 R 또는 다수 P 1R 조합)
//      각 P–R 쌍에 대해 의존성을 카운트한다:
//        - 이미 동일한 from→to 관계가 있으면 Count++
//        - 없으면 새로운 DependencyInfo{To, Count=1, InterfaceType}를 생성한다.
//   4) 결과를 map[from][]DependencyInfo 형태로 변환하여 반환한다.
func ExtractDependenciesAggregatedFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
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

		deMap[deOp] = append(deMap[deOp], portInfo{
			component:     component,
			portType:      portType,
			interfaceType: interfaceType,
		})
	}

	countMap := make(map[string]map[string]*DependencyInfo)

	// deOp 단위 집계
	for _, ports := range deMap {
		var providers []portInfo
		var receivers []portInfo

		for _, p := range ports {
			switch p.portType {
			case "P":
				providers = append(providers, p)
			case "R":
				receivers = append(receivers, p)
			}
		}

		if len(providers) == 0 || len(receivers) == 0 {
			continue
		}

		switch {
		// 1 P, N R
		case len(providers) == 1 && len(receivers) >= 1:
			p := providers[0]
			for _, r := range receivers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					continue
				}

				from := p.component
				to := r.component

				if _, ok := countMap[from]; !ok {
					countMap[from] = make(map[string]*DependencyInfo)
				}
				if existing, ok := countMap[from][to]; ok {
					existing.Count++
				} else {
					countMap[from][to] = &DependencyInfo{
						To:            to,
						Count:         1,
						InterfaceType: p.interfaceType,
					}
				}
			}

		// N P, 1 R
		case len(receivers) == 1 && len(providers) >= 1:
			r := receivers[0]
			for _, p := range providers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					continue
				}

				from := p.component
				to := r.component

				if _, ok := countMap[from]; !ok {
					countMap[from] = make(map[string]*DependencyInfo)
				}
				if existing, ok := countMap[from][to]; ok {
					existing.Count++
				} else {
					countMap[from][to] = &DependencyInfo{
						To:            to,
						Count:         1,
						InterfaceType: p.interfaceType,
					}
				}
			}

		default:
			continue
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

/*
AnalyzeSWCDependencies 함수는 ASW CSV 파일을 입력으로 받아 SWC 간 의존성을 분석하고,
LDI XML(ldi.xml)을 생성하는 상위 레벨 진입점이다.

프로세스:
1) ExtractDependenciesAggregatedFromASW를 호출하여 CSV의 포트 연결을 분석/집계합니다.
   map[from][]DependencyInfo(각 DependencyInfo에는 대상 컴포넌트 To,
   연결 횟수 Count, 인터페이스 유형 InterfaceType이 포함됨)를 얻습니다.
2) 집계 결과를 순회하며 구성:
   - depMap       : map[string][]string        // from → 의존하는 목표 컴포넌트 목록
   - strengthMap  : map[string]map[string]int  // from → (to → 의존 강도/횟수)
3) LDI_Create.GenerateLDIXml(depMap, strengthMap)를 호출하여 LDI XML을 생성합니다.
*/
func AnalyzeSWCDependencies(filePath string) error {
	dependencies, err := ExtractDependenciesAggregatedFromASW(filePath)
	if err != nil {
		return err
	}

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
