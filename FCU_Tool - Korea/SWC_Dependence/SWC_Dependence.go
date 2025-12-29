package SWC_Dependence

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"FCU_Tools/LDI_Create"
)

// 이 구조체는 의존 관계 정보를 저장합니다.
type DependencyInfo struct {
	To            string   //의존 대상 컴포넌트명
	Count         int	   //의존 강도(연결된 링크/선의 개수)
	InterfaceType string   //P 포트인지 R 포트인지(제공/수신 여부)를 기록합니다.
}

// asw 파일을 2차원 배열로 변환하여 rows에 저장한 뒤 반환합니다.
func loadASWRowsFromCSV(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("CSV 파일 열기 실패: %v", err)
	}
	defer f.Close()
	// CSV 읽기용 리더를 생성합니다.
	r := csv.NewReader(f)
	// 각 행의 열 개수가 서로 달라도 허용합니다.
	r.FieldsPerRecord = -1
	//asw 파일의 내용을 rows에 저장합니다. rows는 2차원 배열입니다.
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 행 읽기 실패: %v", err)
	}
	return rows, nil
}

// M3/M4/M6 사용: 각 연결은 독립된 상태로 처리되며, 카운터는 항상 1로 고정됩니다.
// 여기서는 로드된 rows(2차원 배열)를 읽어 map에 저장한 뒤, 관계 분석을 수행합니다.
func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component     string	//컴포넌트 이름
		portType      string	//P 포트인지 R 포트인지 구분
		interfaceType string	//CS인지 SR인지
	}
	//map 생성
	deMap := make(map[string][]portInfo)
	for i, row := range rows {
		// i == 0(헤더 행)이거나 len(row) < 12(열 수가 12 미만)인 경우 해당 행은 건너뜁니다.
		if i == 0 || len(row) < 12 {
			continue
		}
		component := strings.TrimSpace(row[3])
		portType := strings.TrimSpace(row[6])
		interfaceType := strings.TrimSpace(row[8])
		deOp := strings.TrimSpace(row[11])
		//데이터 정제 과정에서 해당 정보가 누락된 행은 제거(버림)됩니다.
		if component == "" || portType == "" || deOp == "" {
			continue
		}
		//Map을 deOp 기준으로 분류하며, 최종 결과는 아래와 같은 형태입니다.
		// deMap["D1"] = []portInfo{
 		// 	{component: "EngineCtrl", portType: "P", interfaceType: "IF_CAN"},
  		// 	{component: "BrakeCtrl",  portType: "R", interfaceType: "IF_CAN"},
  		// 	{component: "DashBoard",  portType: "R", interfaceType: "IF_CAN"},
		// }
		deMap[deOp] = append(deMap[deOp], portInfo{
			component:     component,
			portType:      portType,
			interfaceType: interfaceType,
		})
	}
	//결과 map 생성
	result := make(map[string][]DependencyInfo)

	// deMap의 각 그룹(카테고리)에서 P/R 인터페이스로 분류합니다.
	for _, ports := range deMap {
		var providers []portInfo	//P 인터페이스는 여기에 넣습니다.
		var receivers []portInfo	//R 인터페이스는 여기에 넣습니다.

		// P/R 분리
		for _, p := range ports {
			switch p.portType {
			case "P":
				providers = append(providers, p)
			case "R":
				receivers = append(receivers, p)
			}
		}

		// P 또는 R 중 하나라도 없으면 해당 경우(그룹)는 건너뜁니다.
		if len(providers) == 0 || len(receivers) == 0 {
			continue
		}

		switch {
		// P 인터페이스 1개, R 인터페이스 N개 (1→N)
		case len(providers) == 1 && len(receivers) >= 1:
			p := providers[0]
			for _, r := range receivers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					// 자기 자신에 대한 의존(자기 의존)은 건너뜁니다.
					continue
				}
				result[p.component] = append(result[p.component], DependencyInfo{
					To:            r.component,
					Count:         1,
					InterfaceType: p.interfaceType,
				})
			}

		// P 인터페이스 N개, R 인터페이스 1개 (N→1)
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
			//최종 결과는 아래와 같으며, 시작점에서 도착점으로 이어지는 관계가 생성됩니다.
			// result = map[string][]DependencyInfo{
  			// 	"EngineCtrl": {
    		// 		{To:"BrakeCtrl",  Count:1, InterfaceType:"IF_CAN"},
    		// 		{To:"DashBoard",  Count:1, InterfaceType:"IF_CAN"},
 			// 	},
			// }
		// 이론적으로 P 인터페이스 N개와 R 인터페이스 N개가 동시에 존재하는 경우는 없으므로, 해당 경우는 건너뜁니다.
		default:
			continue
		}
	}

	return result, nil
}

// 이 함수는 위의 함수와 유사하지만, 컴포넌트 간 연결이 여러 개 존재할 경우 Count 값을 누적(증가)합니다. 
// 반면 위의 함수는 Count를 항상 1로 고정하여 합산(집계)하지 않습니다.
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

// AnalyzeSWCDependencies는 ASW.csv 파일의 내용을 LDI.xml 파일로 변환합니다.
func AnalyzeSWCDependencies(filePath string) {
	if strings.TrimSpace(filePath) == "" {
		//Public_data.ConnectorFilePath가 비어 있으면 실패합니다.
		fmt.Println("의존 관계 분석 실패: asw.csv 경로가 비어 있습니다.")
		return
	}

	dependencies, err := ExtractDependenciesAggregatedFromASW(filePath)
	if err != nil {
		fmt.Println("의존 관계 분석 실패:", err)
		return
	}

	// 기존에 ExtractDependenciesAggregatedFromASW로 집계(aggregation)된 정보를 분해합니다.
	// depMap은 의존 관계(누가 누구를 가리키는지)만 저장합니다.
	// strengthMap은 의존(연결) 횟수만 저장합니다.
	depMap := make(map[string][]string)
	strengthMap := make(map[string]map[string]int)
	
	// 여기에서 ExtractDependenciesAggregatedFromASW로 집계된 결과를 분해합니다.
	for from, deps := range dependencies {
		for _, dep := range deps {
			depMap[from] = append(depMap[from], dep.To)
			if strengthMap[from] == nil {
				strengthMap[from] = make(map[string]int)
			}
			strengthMap[from][dep.To] = dep.Count
		}
	}

	// LDI_Create의 LDIXML 생성 함수를 호출하여 LDI를 생성합니다.
	if err := LDI_Create.GenerateLDIXml(depMap, strengthMap); err != nil {
		fmt.Println("의존관계 분석 실패:", err)
		return
	}

	fmt.Println("의존관계 분석 완료.")
}
