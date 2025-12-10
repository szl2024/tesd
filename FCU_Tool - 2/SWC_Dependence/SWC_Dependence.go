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

// normalizeHeader 将表头字符串转换为统一格式，便于忽略大小写和分隔符。
//
// 规则：
//   1) 去掉两端空格。
//   2) 转为小写。
//   3) 去掉空格和下划线。
//
// 例如：
//   "Component"    -> "component"
//   "Port Type"    -> "porttype"
//   "DE_OP"        -> "deop"
func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// detectASWColumnIndices 根据表头自动检测 Component / Port Type / DE_OP 的列索引。
// 要求：
//   - 表头中必须存在：component, port type, de_op（忽略大小写和空格/下划线）。
//   - 若找不到对应列，则返回错误。
func detectASWColumnIndices(rows [][]string) (compIdx, portTypeIdx, deOpIdx int, err error) {
	compIdx, portTypeIdx, deOpIdx = -1, -1, -1

	if len(rows) == 0 {
		err = fmt.Errorf("CSV가 비어 있습니다(행이 없습니다)")
		return
	}

	header := rows[0]
	for i, col := range header {
		key := normalizeHeader(col)
		switch key {
		case "component":
			compIdx = i
		case "porttype":
			portTypeIdx = i
		case "deop":
			deOpIdx = i
		}
	}

	var missing []string
	if compIdx == -1 {
		missing = append(missing, "Component")
	}
	if portTypeIdx == -1 {
		missing = append(missing, "Port Type")
	}
	if deOpIdx == -1 {
		missing = append(missing, "DE_OP")
	}

	if len(missing) > 0 {
		err = fmt.Errorf("CSV 헤더에서 다음 컬럼을 찾을 수 없습니다 (대소문자/공백/밑줄 무시): %v", missing)
	}
	return
}

//  M3/M6 사용: 각 연결은 독립적으로 유지되며, Count는 고정값 1이다.
//  행/열 인덱스를 하드코딩하지 않고, 헤더에서 Component / Port Type / DE_OP 컬럼을 자동 탐지한다.
//  ★ deOp 단위로 1P–N R / N P–1R 관계를 모두 전개한다.
func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	// 1) 从表头自动检测列索引
	compIdx, portTypeIdx, deOpIdx, err := detectASWColumnIndices(rows)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component string
		portType  string
	}

	deMap := make(map[string][]portInfo)

	for i, row := range rows {
		// i == 0 : 헤더 행은 스킵
		if i == 0 {
			continue
		}

		// 如果这一行列数不够访问到需要的列，就跳过这一行
		if len(row) <= compIdx || len(row) <= portTypeIdx || len(row) <= deOpIdx {
			continue
		}

		component := strings.TrimSpace(row[compIdx])
		portType := strings.TrimSpace(row[portTypeIdx])
		deOp := strings.TrimSpace(row[deOpIdx])

		if component == "" || portType == "" || deOp == "" {
			continue
		}

		deMap[deOp] = append(deMap[deOp], portInfo{
			component: component,
			portType:  portType,
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
					InterfaceType: "", // 현재는 interfaceType 사용 안 함
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
					InterfaceType: "",
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
//   2) 헤더에서 Component / Port Type / DE_OP 컬럼 인덱스를 자동 탐지한다.
//   3) 각 행에서 component / portType(P/R) / deOp(연결 식별자)를 추출하여,
//      임시 테이블 deMap[deOp] = []portInfo 형태로 저장한다.
//   4) deOp 단위로 그룹화하여 1P–N R 또는 N P–1R 관계를 모두 전개하고,
//      각 P–R 쌍에 대해 의존성을 카운트한다:
//        - 이미 동일한 from→to 관계가 있으면 Count++
//        - 없으면 새로운 DependencyInfo{To, Count=1}를 생성한다.
//   5) 결과를 map[from][]DependencyInfo 형태로 변환하여 반환한다.
func ExtractDependenciesAggregatedFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	// 1) 从表头自动检测列索引
	compIdx, portTypeIdx, deOpIdx, err := detectASWColumnIndices(rows)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component string
		portType  string
	}

	deMap := make(map[string][]portInfo)

	for i, row := range rows {
		// 헤더 행은 스킵
		if i == 0 {
			continue
		}

		// 如果这一行列数不够访问到需要的列，就跳过
		if len(row) <= compIdx || len(row) <= portTypeIdx || len(row) <= deOpIdx {
			continue
		}

		component := strings.TrimSpace(row[compIdx])
		portType := strings.TrimSpace(row[portTypeIdx])
		deOp := strings.TrimSpace(row[deOpIdx])

		if component == "" || portType == "" || deOp == "" {
			continue
		}

		deMap[deOp] = append(deMap[deOp], portInfo{
			component: component,
			portType:  portType,
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
						InterfaceType: "",
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
						InterfaceType: "",
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
