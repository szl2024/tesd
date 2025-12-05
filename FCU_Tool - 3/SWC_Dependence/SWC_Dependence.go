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

// 读取 ASW CSV 为 [][]string
func loadASWRowsFromCSV(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("CSV 文件打开失败: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// 每行列数可以不一致
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 行读取失败: %v", err)
	}
	return rows, nil
}

// 规范化表头，方便忽略大小写/空格/下划线
//
// 规则：
//   1) 去首尾空格
//   2) 转小写
//   3) 去掉空格和下划线
//
// 示例：
//   "Component"      -> "component"
//   "Port Type"      -> "porttype"
//   "Interface Name" -> "interfacename"
//   "DE_OP"          -> "deop"
func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// 根据表头自动检测 Component / Port Type / Interface Name 的列索引
func detectASWColumnIndices(rows [][]string) (compIdx, portTypeIdx, ifaceNameIdx int, err error) {
	compIdx, portTypeIdx, ifaceNameIdx = -1, -1, -1

	if len(rows) == 0 {
		err = fmt.Errorf("CSV 为空（没有任何行）")
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
		case "interfacename":
			ifaceNameIdx = i
		}
	}

	var missing []string
	if compIdx == -1 {
		missing = append(missing, "Component")
	}
	if portTypeIdx == -1 {
		missing = append(missing, "Port Type")
	}
	if ifaceNameIdx == -1 {
		missing = append(missing, "Interface Name")
	}

	if len(missing) > 0 {
		err = fmt.Errorf("在 CSV 表头中找不到这些列（忽略大小写/空格/下划线）：%v", missing)
	}
	return
}

// ExtractDependenciesRawFromASW
//
// M3/M6 使用：保留“每一条连接”，Count 固定为 1
// 现在：
//   - 不再按 DE_OP 分组
//   - 改为按 “Interface Name” 分组，同名接口的一组端口里找 P/R
func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	// 1) 根据表头自动找列
	compIdx, portTypeIdx, ifaceNameIdx, err := detectASWColumnIndices(rows)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component string
		portType  string
	}

	// key: Interface Name，同名接口归为一组
	ifMap := make(map[string][]portInfo)

	for i, row := range rows {
		// 第 1 行是表头，跳过
		if i == 0 {
			continue
		}

		// 这一行列数不够访问对应列，跳过
		if len(row) <= compIdx || len(row) <= portTypeIdx || len(row) <= ifaceNameIdx {
			continue
		}

		component := strings.TrimSpace(row[compIdx])
		portType := strings.TrimSpace(row[portTypeIdx])
		ifName := strings.TrimSpace(row[ifaceNameIdx])

		if component == "" || portType == "" || ifName == "" {
			continue
		}

		ifMap[ifName] = append(ifMap[ifName], portInfo{
			component: component,
			portType:  portType,
		})
	}

	result := make(map[string][]DependencyInfo)

	// 对每个 Interface Name 组：找出 P / R，生成 from→to 依赖
	for _, ports := range ifMap {
		var pComp, rComp string
		for _, p := range ports {
			if p.portType == "P" {
				pComp = p.component
			} else if p.portType == "R" {
				rComp = p.component
			}
		}
		if pComp != "" && rComp != "" && pComp != rComp {
			result[pComp] = append(result[pComp], DependencyInfo{
				To:            rComp,
				Count:         1,
				InterfaceType: "", // 你已经去掉了 interfaceType 的使用，这里留空
			})
		}
	}

	return result, nil
}

// ExtractDependenciesAggregatedFromASW
//
// 和上面类似，但会把同一对组件 (from, to) 的多条连接合并计数。
// 分组键同样改为 Interface Name。
func ExtractDependenciesAggregatedFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	// 1) 根据表头自动找列
	compIdx, portTypeIdx, ifaceNameIdx, err := detectASWColumnIndices(rows)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component string
		portType  string
	}

	ifMap := make(map[string][]portInfo)

	for i, row := range rows {
		// 表头行跳过
		if i == 0 {
			continue
		}

		// 列数不够，跳过
		if len(row) <= compIdx || len(row) <= portTypeIdx || len(row) <= ifaceNameIdx {
			continue
		}

		component := strings.TrimSpace(row[compIdx])
		portType := strings.TrimSpace(row[portTypeIdx])
		ifName := strings.TrimSpace(row[ifaceNameIdx])

		if component == "" || portType == "" || ifName == "" {
			continue
		}

		ifMap[ifName] = append(ifMap[ifName], portInfo{
			component: component,
			portType:  portType,
		})
	}

	// countMap[from][to] = *DependencyInfo
	countMap := make(map[string]map[string]*DependencyInfo)

	for _, ports := range ifMap {
		var pComp, rComp string
		for _, p := range ports {
			if p.portType == "P" {
				pComp = p.component
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
					InterfaceType: "", // 不再使用接口类型
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

/*
AnalyzeSWCDependencies 是上层入口：

1) 调用 ExtractDependenciesAggregatedFromASW 分析/聚合依赖：
   得到 map[from][]DependencyInfo
2) 转成：
   - depMap[from] = []to
   - strengthMap[from][to] = Count
3) 调用 LDI_Create.GenerateLDIXml(depMap, strengthMap) 生成 ldi.xml
*/
func AnalyzeSWCDependencies(filePath string) error {
	dependencies, err := ExtractDependenciesAggregatedFromASW(filePath)
	if err != nil {
		return err
	}

	// 转为生成 LDI 需要的格式
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

	// 生成 ldi.xml
	err = LDI_Create.GenerateLDIXml(depMap, strengthMap)
	if err != nil {
		return fmt.Errorf("LDI 文件生成失败: %v", err)
	}

	fmt.Println("✅ LDI 文件生成完成.")
	return nil
}
