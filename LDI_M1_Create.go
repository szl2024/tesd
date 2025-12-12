package LDI_M1_Create

import (
	"encoding/csv"
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

// buildRunnableToModelMap
// 从 asw.csv 中构建 runnable → 模型名 的映射。
// 约定：
//   - 第 4 列 (index 3): 模型名
//   - 第 6 列 (index 5): runnable 名
func buildRunnableToModelMap() (map[string]string, error) {
	result := make(map[string]string)

	csvPath := Public_data.ConnectorFilePath
	if csvPath == "" {
		// 没有设置 asw.csv 路径，则返回空映射，后面会直接使用原始名字
		return result, nil
	}

	f, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("打开 asw.csv 失败（ConnectorFilePath = %s）: %v", csvPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// 允许每行列数不同
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取 asw.csv 内容失败: %v", err)
	}

	// 默认第一行为表头，从第二行开始
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) <= 5 {
			continue
		}

		modelName := strings.TrimSpace(row[3]) // 第 4 列：模型名
		runnable := strings.TrimSpace(row[5]) // 第 6 列：runnable 名

		if modelName == "" || runnable == "" {
			continue
		}

		// 一个 runnable 对应一个模型，重复时保持第一次或最后一次均可，这里覆盖即可
		result[runnable] = modelName
	}

	return result, nil
}

// mapM1ElementName
// 把 M1 LDI 中的 element.Name / uses.Provider 从 runnable 名映射为模型名。
// 规则：
//   - 从名字中取前缀：runnablePart = name 的第一个 '.' 之前部分（或者整个 name，如果没有 '.'）
//   - 若 runnablePart 在 runnable→model 映射中存在：
//         newName = modelName + suffix（suffix 为原 name 去掉 runnablePart 后的剩余部分，包括 '.'）
//     例如：
//         name = "RCL1Cm1_Te10"                  → "CL1CM1"
//         name = "RCL1Cm1_Te10.CL1CM1CLS1"       → "CL1CM1.CL1CM1CLS1"
//   - 若不存在映射，则保持原 name 不变。
func mapM1ElementName(name string, runnableToModel map[string]string) string {
	if len(runnableToModel) == 0 {
		return name
	}

	runnablePart := name
	suffix := ""

	if idx := strings.Index(name, "."); idx >= 0 {
		runnablePart = name[:idx]
		suffix = name[idx:] // 包含 '.'
	}

	if model, ok := runnableToModel[runnablePart]; ok && model != "" {
		return model + suffix
	}

	return name
}

// RewriteM1LDIFilesRename
// 直接修改 M1_Public_Data.LDIDir 下所有 *.ldi.xml：
//   - <element name="..."> 里的 name
//   - <uses provider="..."> 里的 provider
// 按 asw.csv 的 runnable→模型名 映射进行就地替换并写回原文件。
func RewriteM1LDIFilesRename(runnableToModel map[string]string) error {
	if len(runnableToModel) == 0 {
		// 没有映射就不改任何文件
		return nil
	}

	m1Dir := M1_Public_Data.LDIDir
	if m1Dir == "" {
		return fmt.Errorf("M1_Public_Data.LDIDir 未设置，无法找到 M1 的 LDI 文件目录")
	}

	entries, err := os.ReadDir(m1Dir)
	if err != nil {
		return fmt.Errorf("读取 M1 LDI 目录失败 [%s]: %v", m1Dir, err)
	}

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
			return fmt.Errorf("读取 M1 LDI 文件失败 [%s]: %v", path, err)
		}

		var root Root
		if err := xml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("解析 M1 LDI XML 失败 [%s]: %v", path, err)
		}

		changed := false

		for i := range root.Items {
			// element name
			oldName := root.Items[i].Name
			newName := mapM1ElementName(oldName, runnableToModel)
			if newName != oldName {
				root.Items[i].Name = newName
				changed = true
			}

			// uses provider
			for j := range root.Items[i].Uses {
				oldProv := root.Items[i].Uses[j].Provider
				newProv := mapM1ElementName(oldProv, runnableToModel)
				if newProv != oldProv {
					root.Items[i].Uses[j].Provider = newProv
					changed = true
				}
			}
		}

		if !changed {
			continue
		}

		out, err := xml.MarshalIndent(root, "  ", "    ")
		if err != nil {
			return fmt.Errorf("序列化 M1 LDI XML 失败 [%s]: %v", path, err)
		}

		header := []byte(xml.Header)
		if err := ioutil.WriteFile(path, append(header, out...), 0644); err != nil {
			return fmt.Errorf("写回 M1 LDI 文件失败 [%s]: %v", path, err)
		}
	}

	return nil
}

// MergeM1ToMainLDI
// 将 M1 阶段在 LDIDir 目录下生成的所有 *.ldi.xml 中的 coverage.m1
// 合并到主 LDI (Output/result.ldi.xml) 中。
//
// 额外步骤：
//   1) 利用 asw.csv (Public_data.ConnectorFilePath) 中的 runnable → 模型名映射
//   2) 先就地修改 M1 的 *.ldi.xml（element name / uses provider）为“模型名”
//   3) 再把 M1 LDI 中的 coverage.m1 合并到主 LDI
func MergeM1ToMainLDI() error {
	// 1) 确定主 LDI 路径
	if Public_data.OutputDir == "" {
		return fmt.Errorf("主 LDI 输出目录未初始化，请先调用 InitOutputDirectory")
	}
	mainLDIPath := filepath.Join(Public_data.OutputDir, "result.ldi.xml")

	// 2) 读取 runnable → 模型名 映射
	runnableToModel, err := buildRunnableToModelMap()
	if err != nil {
		// 构建失败时给出提示，但为了不完全阻塞，也可以选择直接返回错误
		return fmt.Errorf("构建 runnable → 模型名 映射失败: %v", err)
	}

	// 2.1) 先把 M1 的 *.ldi.xml 直接改名（写回文件）
	if err := RewriteM1LDIFilesRename(runnableToModel); err != nil {
		return err
	}

	// 3) 读取主 LDI
	mainData, err := ioutil.ReadFile(mainLDIPath)
	if err != nil {
		return fmt.Errorf("读取主 LDI 文件失败 [%s]: %v", mainLDIPath, err)
	}

	var mainRoot Root
	if err := xml.Unmarshal(mainData, &mainRoot); err != nil {
		return fmt.Errorf("解析主 LDI XML 失败: %v", err)
	}

	// 4) 扫描 M1 LDI 目录，收集 coverage.m1 （Key 为“映射后的名字”）
	m1Dir := M1_Public_Data.LDIDir
	if m1Dir == "" {
		return fmt.Errorf("M1_Public_Data.LDIDir 未设置，无法找到 M1 的 LDI 文件目录")
	}

	entries, err := os.ReadDir(m1Dir)
	if err != nil {
		return fmt.Errorf("读取 M1 LDI 目录失败 [%s]: %v", m1Dir, err)
	}

	m1Map := make(map[string]string) // name(已映射为模型名) → coverage.m1

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
					// 兼容：即使前面没成功写回，这里仍然再映射一次
					mappedName := mapM1ElementName(el.Name, runnableToModel)
					m1Map[mappedName] = p.Value
				}
			}
		}
	}

	if len(m1Map) == 0 {
		fmt.Println("ℹ️ 在 M1 LDI 目录中未找到任何 coverage.m1 属性，不修改主 LDI。")
		return nil
	}

	// 5) 建立主 LDI 的索引表：element name → index
	mainIndex := make(map[string]int)
	for i, el := range mainRoot.Items {
		mainIndex[el.Name] = i
	}

	// 6) 合并 coverage.m1 到主 LDI
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

	// 7) 写回主 LDI
	out, err := xml.MarshalIndent(mainRoot, "  ", "    ")
	if err != nil {
		return fmt.Errorf("主 LDI XML 序列化失败: %v", err)
	}

	header := []byte(xml.Header)
	if err := ioutil.WriteFile(mainLDIPath, append(header, out...), 0644); err != nil {
		return fmt.Errorf("写回主 LDI 文件失败 [%s]: %v", mainLDIPath, err)
	}

	fmt.Println("✅ M1 指标 coverage.m1 已根据 asw.csv 的模型名成功合并到主 LDI（包含新增元素）")
	return nil
}
