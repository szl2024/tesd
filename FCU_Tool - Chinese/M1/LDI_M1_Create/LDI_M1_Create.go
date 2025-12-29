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

// XML 结构定义
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

// MergeM1ToMainLDI
// 将 M1 阶段在 LDIDir 目录下生成的所有 *.ldi.xml 中的 coverage.m1
// 合并到主 LDI (Output/result.ldi.xml) 中。
//
// 说明：
// - 假设 M1 的 *.ldi.xml 在生成阶段已完成“模型名换名”（例如由 GenerateM1LDIFromTxt 以 txt 文件名作为模型名）
// - 因此这里不再读取 asw.csv，也不再做 runnable→模型名映射，更不会就地改写 M1 的 ldi.xml。
func MergeM1ToMainLDI() error {
	// 1) 确定主 LDI 路径
	if Public_data.OutputDir == "" {
		return fmt.Errorf("主 LDI 输出目录未初始化，请先调用 InitOutputDirectory")
	}
	mainLDIPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")

	// 2) 读取主 LDI
	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("读取主 LDI 文件失败 [%s]: %v", mainLDIPath, err)
	}

	var mainRoot Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("解析主 LDI XML 失败: %v", err)
	}

	// 3) 扫描 M1 LDI 目录，收集 coverage.m1（Key 直接使用 el.Name）
	m1Dir := M1_Public_Data.LDIDir
	if m1Dir == "" {
		return fmt.Errorf("M1_Public_Data.LDIDir 未设置，无法找到 M1 的 LDI 文件目录")
	}

	entries, err := os.ReadDir(m1Dir)
	if err != nil {
		return fmt.Errorf("读取 M1 LDI 目录失败 [%s]: %v", m1Dir, err)
	}

	m1Map := make(map[string]string) // element name → coverage.m1

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".ldi.xml") {
			continue
		}

		path := filepath.Join(m1Dir, e.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("⚠️ 读取 M1 LDI 文件失败 [%s]: %v\n", path, err)
			continue
		}

		var m1Root Root
		if err := xml.Unmarshal(data, &m1Root); err != nil {
			fmt.Printf("⚠️ 解析 M1 LDI 文件失败 [%s]: %v\n", path, err)
			continue
		}

		for _, el := range m1Root.Items {
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					m1Map[el.Name] = p.Value
				}
			}
		}
	}

	if len(m1Map) == 0 {
		fmt.Println("ℹ️ 在 M1 LDI 目录中未找到任何 coverage.m1 属性，不修改主 LDI。")
		return nil
	}

	// 4) 建立主 LDI 的索引表：element name → index
	mainIndex := make(map[string]int)
	for i, el := range mainRoot.Items {
		mainIndex[el.Name] = i
	}

	// 5) 合并 coverage.m1 到主 LDI
	for name, val := range m1Map {
		if idx, ok := mainIndex[name]; ok {
			// 主 LDI 已有该 element：检查是否已有 coverage.m1
			el := &mainRoot.Items[idx]
			exists := false
			for _, p := range el.Property {
				if p.Name == "coverage.m1" {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			el.Property = append(el.Property, Property{
				Name:  "coverage.m1",
				Value: val,
			})
		} else {
			// 主 LDI 尚无该 element：新增一个 element，仅带 coverage.m1
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

	fmt.Println("✅ M1 指标 coverage.m1 已成功合并到主 LDI")
	return nil
}
