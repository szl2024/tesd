package SWC_Dependence

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"FCU_Tools/LDI_Create"
)

// 这个结构体记录了依赖关系的信息
type DependencyInfo struct {
	To            string   //依赖目标组件名 
	Count         int	   //依赖强度(有多少条线链接了)
	InterfaceType string   //记录了P - port还是 R - Port，提供还是接收
}

// 将asw文件变为一个二维数组存储到rows中并返回
func loadASWRowsFromCSV(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("CSV文件打开失败: %v", err)
	}
	defer f.Close()
	// 创建 CSV 读取器
	r := csv.NewReader(f)
	// 允许每行列数不一致
	r.FieldsPerRecord = -1
	//将asw文件的内容存储到rows中，这个是一个二维数组
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 行读取失败: %v", err)
	}
	return rows, nil
}

// M3/M6 使用：每个连接均为独立状态，计数器固定为1。
// 这里将得到的rows的二维数组进行读取，并将其放入map中，进行关系的分析。
func ExtractDependenciesRawFromASW(filePath string) (map[string][]DependencyInfo, error) {
	rows, err := loadASWRowsFromCSV(filePath)
	if err != nil {
		return nil, err
	}

	type portInfo struct {
		component     string	//组件名
		portType      string	//P 还是 R port
		interfaceType string	//CS还是 SR
	}
	//创建map
	deMap := make(map[string][]portInfo)
	for i, row := range rows {
		// i == 0 表头 或 len(row) < 12 : 每一行中如果小于12列那就跳过
		if i == 0 || len(row) < 12 {
			continue
		}
		component := strings.TrimSpace(row[3])
		portType := strings.TrimSpace(row[6])
		interfaceType := strings.TrimSpace(row[8])
		deOp := strings.TrimSpace(row[11])
		//数据清洗，将会丢掉缺失这些信息的行
		if component == "" || portType == "" || deOp == "" {
			continue
		}
		//将Map 按照deOp来进行分类，最终结果类似如下
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
	//创建结果map
	result := make(map[string][]DependencyInfo)

	// 将demap中每个类别进行P / R接口的分类
	for _, ports := range deMap {
		var providers []portInfo	//P接口放入这里
		var receivers []portInfo	//R接口放入这里

		// P / R 分离
		for _, p := range ports {
			switch p.portType {
			case "P":
				providers = append(providers, p)
			case "R":
				receivers = append(receivers, p)
			}
		}

		// P或者R 一个都不存在的情况就会跳过
		if len(providers) == 0 || len(receivers) == 0 {
			continue
		}

		switch {
		// 1个P接口, N个R接口 (1→N)
		case len(providers) == 1 && len(receivers) >= 1:
			p := providers[0]
			for _, r := range receivers {
				if p.component == "" || r.component == "" {
					continue
				}
				if p.component == r.component {
					// 自己依赖自己就会跳过
					continue
				}
				result[p.component] = append(result[p.component], DependencyInfo{
					To:            r.component,
					Count:         1,
					InterfaceType: p.interfaceType,
				})
			}

		// N个P接口, 1个R接口 (N→1)
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
			//最终结果如下，将会生成这样的从开始到到达的关系
			// result = map[string][]DependencyInfo{
  			// 	"EngineCtrl": {
    		// 		{To:"BrakeCtrl",  Count:1, InterfaceType:"IF_CAN"},
    		// 		{To:"DashBoard",  Count:1, InterfaceType:"IF_CAN"},
 			// 	},
			// }
		// 理论上不存在N个R接口和N个P接口，将这部分跳过
		default:
			continue
		}
	}

	return result, nil
}

// 这个函数跟上面的函数类似，只不过如果出现组件与组件间有多个连接，则会将Count的计数增加。而上面的是固定为1，不进行汇总
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

//AnalyzeSWCDependencies将ASW.CSV文件的内容转化为LDI.XML文件
func AnalyzeSWCDependencies(filePath string) {
	if strings.TrimSpace(filePath) == "" {
		//数据 Public_data.ConnectorFilePath为空，会造成失败
		fmt.Println("依赖关系分析失败: asw.csv 路径为空.")
		return
	}

	dependencies, err := ExtractDependenciesAggregatedFromASW(filePath)
	if err != nil {
		fmt.Println("依赖关系分析失败:", err)
		return
	}

	// 将原来的通过ExtractDependenciesAggregatedFromASW聚合得到的信息进行拆分
	//depMap只存储依赖关系，谁指向谁
	//strengthMap只存储依赖的次数。
	depMap := make(map[string][]string)
	strengthMap := make(map[string]map[string]int)
	
	//通过这里进行将ExtractDependenciesAggregatedFromASW聚合得到的信息进行拆分
	for from, deps := range dependencies {
		for _, dep := range deps {
			depMap[from] = append(depMap[from], dep.To)
			if strengthMap[from] == nil {
				strengthMap[from] = make(map[string]int)
			}
			strengthMap[from][dep.To] = dep.Count
		}
	}

	// 调用LDI_Create的LDIXML的生成函数，生成LDI
	if err := LDI_Create.GenerateLDIXml(depMap, strengthMap); err != nil {
		fmt.Println("의존관계 분석 실패:", err)
		return
	}

	fmt.Println("의존관계 분석 완료.")
}
