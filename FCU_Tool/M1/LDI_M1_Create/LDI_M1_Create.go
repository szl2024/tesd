package LDI_M1_Create

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/Public_data"
)

// MergeM1ToMainLDI
// 将 M1 阶段在 LDIDir 目录下生成的所有 *.ldi.xml 中的 coverage.m1
// 合并写入到主结果文件 result.ldi.xml 中。
//
// 逻辑：
//   1) 读取主 LDI 文件：Public_data.OutputDir/result.ldi.xml
//   2) 枚举 M1_Public_Data.LDIDir 目录下所有 *.ldi.xml
//      逐个解析，收集其中的 coverage.m1 指标，按 element name 建立 m1Map[name] = value
//      （如有重名，后出现的覆盖前面的值，保持简单策略）
//   3) 合并到主 LDI：
//        - 如果主 LDI 中存在同名 element：
//            - 若该 element 尚不存在 coverage.m1 属性，则追加
//        - 如果主 LDI 中不存在同名 element：
//            - 创建新的 element，并写入 coverage.m1 属性
//   4) 将修改后的主 LDI 覆盖写回 result.ldi.xml
func MergeM1ToMainLDI() error {
	// 与 M2 合并逻辑保持一致的局部结构定义
	type Property struct {
		XMLName xml.Name `xml:"property"`
		Name    string   `xml:"name,attr"`
		Value   string   `xml:",chardata"`
	}
	type Uses struct {
		XMLName  xml.Name `xml:"uses"`
		Provider string   `xml:"provider,attr"`
		Strength string   `xml:"strength,attr,omitempty"`
	}
	type Element struct {
		XMLName  xml.Name   `xml:"element"`
		Name     string     `xml:"name,attr"`
		Uses     []Uses     `xml:"uses"`
		Property []Property `xml:"property"`
	}
	type Root struct {
		XMLName xml.Name  `xml:"ldi"`
		Items   []Element `xml:"element"`
	}

	// 1) 主结果 LDI 路径
	mainLDIPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")

	// 2) M1 LDI 文件所在目录
	m1LDIDir := M1_Public_Data.LDIDir
	if m1LDIDir == "" {
		return fmt.Errorf("M1 LDIDir 为空，请先调用 M1_Public_Data.SetWorkDir 初始化工作空间")
	}

	// 读取主 LDI
	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("读取主 LDI 文件失败 [%s]: %v", mainLDIPath, err)
	}

	var mainRoot Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("解析主 LDI XML 失败: %v", err)
	}

	// 3) 扫描 M1 LDIDir 下所有 *.ldi.xml，构建 m1Map
	m1Map := make(map[string]string)

	entries, err := os.ReadDir(m1LDIDir)
	if err != nil {
		return fmt.Errorf("读取 M1 LDIDir 目录失败 [%s]: %v", m1LDIDir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// 只处理 .ldi.xml
		if !strings.HasSuffix(strings.ToLower(name), ".ldi.xml") {
			continue
		}

		ldiPath := filepath.Join(m1LDIDir, name)
		data, err := ioutil.ReadFile(ldiPath)
		if err != nil {
			// 不中断，跳过有问题的文件
			fmt.Printf("⚠️ 读取 M1 LDI 文件失败 [%s]: %v\n", ldiPath, err)
			continue
		}

		var m1Root Root
		if err := xml.Unmarshal(data, &m1Root); err != nil {
			fmt.Printf("⚠️ 解析 M1 LDI XML 失败 [%s]: %v\n", ldiPath, err)
			continue
		}

		for _, el := range m1Root.Items {
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					// 简单策略：如同名 element 多个文件都有，后者覆盖前者
					m1Map[el.Name] = p.Value
				}
			}
		}
	}

	// 如果没有任何 M1 指标，直接返回，不改主文件
	if len(m1Map) == 0 {
		fmt.Println("ℹ️ 未在 M1 LDIDir 中找到任何 coverage.m1 指标，主 LDI 不做修改")
		return nil
	}

	// 4) 先构建主 LDI 的快速索引：element name -> index
	mainIndex := make(map[string]int)
	for i, el := range mainRoot.Items {
		mainIndex[el.Name] = i
	}

	// 5) 将 m1Map 合并回主 LDI
	for name, val := range m1Map {
		if idx, ok := mainIndex[name]; ok {
			// 主 LDI 已存在同名 element，检查 coverage.m1 是否已经存在
			el := &mainRoot.Items[idx]
			alreadyExists := false
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					alreadyExists = true
					break
				}
			}
			if alreadyExists {
				continue
			}

			el.Property = append(el.Property, Property{
				Name:  "coverage.m1",
				Value: val,
			})
		} else {
			// 主 LDI 中不存在同名 element，新建一个 element 写入 coverage.m1
			newEl := Element{
				Name: name,
				Property: []Property{
					{
						Name:  "coverage.m1",
						Value: val,
					},
				},
			}
			mainRoot.Items = append(mainRoot.Items, newEl)
			// 更新索引，避免后续出现同名覆盖问题
			mainIndex[name] = len(mainRoot.Items) - 1
		}
	}

	// 6) 写回主 LDI
	out, err := xml.MarshalIndent(mainRoot, "  ", "    ")
	if err != nil {
		return fmt.Errorf("主 LDI XML 序列化失败: %v", err)
	}

	header := []byte(xml.Header)
	if err := ioutil.WriteFile(mainLDIPath, append(header, out...), 0644); err != nil {
		return fmt.Errorf("写回主 LDI 文件失败 [%s]: %v", mainLDIPath, err)
	}

	fmt.Println("✅ M1 指标 coverage.m1 已成功合并到主 LDI（包含新增元素）")
	return nil
}
