package File_Utils_M1

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// 2. 读取 Windows 路径：控制台提示 + 读入 + 保存到 M1_Public_Data.SrcPath
func ReadWindowsPath() {
	fmt.Print("请输入一个 Windows 路径： ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("读取输入失败：", err)
		return
	}

	input = strings.TrimSpace(input)
	M1_Public_Data.SrcPath = input
}

// 3. 从 SrcPath 下的子文件夹中，复制同名 slx 文件到 BuildDir
//   SrcPath/
//     ├─ ModelA/  →  ModelA/ModelA.slx  复制到  BuildDir/ModelA.slx
//     ├─ ModelB/  →  ModelB/ModelB.slx  复制到  BuildDir/ModelB.slx
// 同时在 TxtDir 下创建同名的 txt 文件：ModelA.txt、ModelB.txt
func CopySlxToBuild() {
	srcRoot := M1_Public_Data.SrcPath
	dstRoot := M1_Public_Data.BuildDir
	txtRoot := M1_Public_Data.TxtDir

	if srcRoot == "" {
		fmt.Println("SrcPath 为空，请先调用 ReadWindowsPath() 输入路径")
		return
	}
	if dstRoot == "" {
		fmt.Println("BuildDir 为空，请先调用 SetWorkDir() 初始化工作空间")
		return
	}
	if txtRoot == "" {
		fmt.Println("TxtDir 为空，请检查 SetWorkDir() 是否正确设置")
		return
	}

	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		fmt.Println("无法读取 SrcPath 目录：", err)
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		folderName := e.Name()
		slxPath := filepath.Join(srcRoot, folderName, folderName+".slx")
		if _, err := os.Stat(slxPath); err != nil {
			// 没有同名 slx，跳过
			continue
		}

		// 目标 slx 文件路径：BuildDir/同名.slx
		dstPath := filepath.Join(dstRoot, folderName+".slx")

		// 复制 slx 文件
		if err := copyFile(slxPath, dstPath); err != nil {
			fmt.Printf("复制失败 [%s] → [%s]：%v\n", slxPath, dstPath, err)
			continue
		}

		// 在 TxtDir 下创建同名 txt 文件
		txtPath := filepath.Join(txtRoot, folderName+".txt")
		f, err := os.Create(txtPath) // 每次运行重建/清空
		if err != nil {
			fmt.Printf("无法创建 txt 文件 [%s]：%v\n", txtPath, err)
			continue
		}
		_ = f.Close()
	}
}

// 4. 解压 BuildDir 下的 slx 文件到同名目录
//   BuildDir/
//     ├─ ModelA.slx  → 解压到 BuildDir/ModelA/.
//     ├─ ModelB.slx  → 解压到 BuildDir/ModelB/.
func UnzipSlxFiles() {
	buildRoot := M1_Public_Data.BuildDir
	if buildRoot == "" {
		fmt.Println("BuildDir 为空，请先调用 SetWorkDir() 初始化工作空间")
		return
	}

	entries, err := os.ReadDir(buildRoot)
	if err != nil {
		fmt.Println("无法读取 BuildDir 目录：", err)
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if strings.ToLower(filepath.Ext(name)) != ".slx" {
			continue
		}

		slxPath := filepath.Join(buildRoot, name)
		modelName := strings.TrimSuffix(name, filepath.Ext(name))
		destDir := filepath.Join(buildRoot, modelName)

		// 确保解压目录是干净的
		_ = os.RemoveAll(destDir)

		if err := unzipOne(slxPath, destDir); err != nil {
			fmt.Printf("解压失败 [%s] → [%s]：%v\n", slxPath, destDir, err)
			continue
		}
	}
}

// 简单的文件复制工具
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// 解压单个 slx(zip) 到 destDir
func unzipOne(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)

		// 目录
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		// 确保上级目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return err
		}

		outFile.Close()
		rc.Close()
	}
	return nil
}

// ===================== M1 LDI 生成相关 =====================

// 用于从 txt 中解析出来的节点信息
type m1Node struct {
	Level          int
	Name           string
	SID            string
	Father         string
	Ports          int     // 当前节点自己的端口个数（包括 virtual port）
	CSPorts        int     // 仅 L1 的 C-S 端口数
	ChildCount     int     // 直接子节点个数
	ChildPorts     int     // 直接子节点端口数之和
	EffectivePorts float64 // L1: 加权端口数; 其他层: 等于 Ports
	Coverage       float64 // 计算出的 m1
}

// LDI XML 结构
type ldiProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",chardata"`
}

type ldiElement struct {
	XMLName  xml.Name      `xml:"element"`
	Name     string        `xml:"name,attr"`
	Property []ldiProperty `xml:"property"`
}

type ldiRoot struct {
	XMLName xml.Name     `xml:"ldi"`
	Items   []ldiElement `xml:"element"`
}

// 6. 根据 TxtDir 下的 txt 文件生成对应的 ldi.xml
//    例如 TurnLight.txt -> TurnLight.ldi.xml
//    规则：如果存在 N 层，只对 1.N-1 层计算并输出 m1，最底层 N 不输出
//    同时在 TxtDir 下生成 XXX_m1.txt
func GenerateM1LDIFromTxt() {
	txtRoot := M1_Public_Data.TxtDir
	ldiRoot := M1_Public_Data.LDIDir

	if txtRoot == "" {
		fmt.Println("TxtDir 为空，请检查 SetWorkDir() 是否正确设置")
		return
	}
	if ldiRoot == "" {
		fmt.Println("LDIDir 为空，请检查 SetWorkDir() 是否正确设置")
		return
	}

	entries, err := os.ReadDir(txtRoot)
	if err != nil {
		fmt.Println("读取 TxtDir 失败:", err)
		return
	}

	// 确保 LDI 目录存在
	if err := os.MkdirAll(ldiRoot, 0755); err != nil {
		fmt.Println("创建 LDI 目录失败:", err)
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.ToLower(filepath.Ext(name)) != ".txt" {
			continue
		}

		txtPath := filepath.Join(txtRoot, name)
		modelName := strings.TrimSuffix(name, filepath.Ext(name))

		nodes, err := parseM1NodesFromTxt(txtPath)
		if err != nil {
			fmt.Printf("解析 txt 失败 [%s]: %v\n", txtPath, err)
			continue
		}
		if len(nodes) == 0 {
			fmt.Printf("txt 中没有解析到节点 [%s]\n", txtPath)
			continue
		}

		computeM1ForNodes(nodes)

		// 生成 ldi.xml
		ldiPath := filepath.Join(ldiRoot, modelName+".ldi.xml")
		if err := writeM1LDI(ldiPath, modelName, nodes); err != nil {
			fmt.Printf("写入 LDI 失败 [%s]: %v\n", ldiPath, err)
			// 不中断，继续生成 m1.txt
		} else {
			//fmt.Printf("已生成 LDI：%s\n", ldiPath)
		}

		// 生成 XXX_m1.txt
		statsPath := filepath.Join(txtRoot, modelName+"_m1.txt")
		if err := writeM1StatsTxt(statsPath, nodes); err != nil {
			fmt.Printf("写入 m1 统计失败 [%s]: %v\n", statsPath, err)
		} else {
			//fmt.Printf("已生成 m1 统计：%s\n", statsPath)
		}
	}
}

// 解析一个 txt，把所有 [Lx] block 和 [Lx Port]/[Lx virtual Port] 全部解析出来
func parseM1NodesFromTxt(txtPath string) ([]*m1Node, error) {
	f, err := os.Open(txtPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	var (
		nodes   []*m1Node
		curNode *m1Node
	)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Block 行：以 "[L" 开头且没有前导 Tab
		if strings.HasPrefix(line, "[L") {
			trim := strings.TrimSpace(line)
			levelRe := regexp.MustCompile(`^\[L(\d+)\]`)
			m := levelRe.FindStringSubmatch(trim)
			if len(m) >= 2 {
				level, name, sid, father, ok := parseBlockLineInfo(trim)
				if !ok {
					continue
				}
				node := &m1Node{
					Level:  level,
					Name:   name,
					SID:    sid,
					Father: father,
				}
				nodes = append(nodes, node)
				curNode = node
				continue
			}
		}

		// 端口行形如：\t[L1 Port] 或 \t[L2 virtual Port]
		if strings.HasPrefix(line, "\t[L") {
			if curNode == nil {
				continue
			}
			trim := strings.TrimLeft(line, "\t")
			endIdx := strings.Index(trim, "]")
			if endIdx <= 0 {
				continue
			}
			tag := trim[:endIdx+1] // [L1 Port] / [L2 virtual Port]
			content := strings.TrimSpace(trim[endIdx+1:])

			// 只统计 Port 数，不关心名称
			// L1 的 C-S 端口在后面会单独解析，这里不加
			if strings.Contains(tag, "Port") {
				curNode.Ports++
				_ = content
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 第二遍：统计每个节点的 childCount / childPorts
	// child: Level = parent.Level + 1 且 child.Father == parent.Name
	for _, p := range nodes {
		for _, c := range nodes {
			if c.Level == p.Level+1 && c.Father != "" && c.Father == p.Name {
				p.ChildCount++
				p.ChildPorts += c.Ports
			}
		}
	}

	// 第三步：统计 L1 的 C-S 端口（在 txt 文件里标记为 “C-S”）
	// 规则：如果一行包含 "C-S" 则认为是 C-S 端口
	// 注意：这里只统计 ports 数，不做复杂 parsing
	{
		// 重新扫一遍 txt
		f2, err := os.Open(txtPath)
		if err == nil {
			defer f2.Close()
			sc := bufio.NewScanner(f2)
			for sc.Scan() {
				ln := sc.Text()
				if strings.Contains(ln, "C-S") {
					// 找到当前的 L1 节点（一般只有一个）
					for _, n := range nodes {
						if n.Level == 1 {
							n.CSPorts++
							break
						}
					}
				}
			}
		}
	}

	// 计算 EffectivePorts
	for _, n := range nodes {
		if n.Level == 1 {
			// L1: effective = ports + 0.5*(childPorts) + C-S
			n.EffectivePorts = float64(n.Ports) + 0.5*float64(n.ChildPorts) + float64(n.CSPorts)
		} else {
			// 其他层: effective = ports
			n.EffectivePorts = float64(n.Ports)
		}
	}

	return nodes, nil
}

func parseBlockLineInfo(line string) (level int, name, sid, father string, ok bool) {
	// line 形如：
	// [L1] Name: RCL1Cm1_Te10	BlockType=SubSystem	SID=1
	// [L2] Name: CL1CM1CLS1	BlockType=SubSystem 	SID=8         	FatherNode=RCL1Cm1_Te10
	levelRe := regexp.MustCompile(`^\[L(\d+)\]`)
	m := levelRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0, "", "", "", false
	}
	lv, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", "", "", false
	}
	level = lv

	// Name
	nameRe := regexp.MustCompile(`Name:\s*([^\t]+)`)
	m2 := nameRe.FindStringSubmatch(line)
	if len(m2) < 2 {
		return 0, "", "", "", false
	}
	name = strings.TrimSpace(m2[1])

	// SID
	sidRe := regexp.MustCompile(`SID=([^\t\s]+)`)
	m3 := sidRe.FindStringSubmatch(line)
	if len(m3) < 2 {
		return 0, "", "", "", false
	}
	sid = strings.TrimSpace(m3[1])

	// FatherNode（可选）
	fatherRe := regexp.MustCompile(`FatherNode=([^\t\s]+)`)
	m4 := fatherRe.FindStringSubmatch(line)
	if len(m4) >= 2 {
		father = strings.TrimSpace(m4[1])
	}

	return level, name, sid, father, true
}

// 计算每个节点的 m1 coverage
func computeM1ForNodes(nodes []*m1Node) {
	for _, n := range nodes {
		n.Coverage = computeM1(n)
	}
}

// m1 公式：m1 = 1 - 1/(1+effectivePorts)
// （这里只给一个示例公式，如果你有真实公式可替换）
func computeM1(n *m1Node) float64 {
	ep := n.EffectivePorts
	return 1.0 - 1.0/(1.0+ep)
}

// 把 nodes 写成一个 m1 统计文件（XXX_m1.txt）
func writeM1StatsTxt(statsPath string, nodes []*m1Node) error {
	f, err := os.Create(statsPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 按 level、name 排序
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Level != nodes[j].Level {
			return nodes[i].Level < nodes[j].Level
		}
		return nodes[i].Name < nodes[j].Name
	})

	// 写表头
	_, _ = f.WriteString("Level\tName\tSID\tFather\tPorts\tCSPorts\tChildCount\tChildPorts\tEffectivePorts\tCoverage\n")

	for _, n := range nodes {
		line := fmt.Sprintf(
			"%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%.4f\t%.4f\n",
			n.Level, n.Name, n.SID, n.Father, n.Ports, n.CSPorts, n.ChildCount, n.ChildPorts, n.EffectivePorts, n.Coverage,
		)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// Name  => L1.Name + "." + L2.Name
// L3: L1.Name + "." + L2.Name + "." + L3.Name
func buildHierNameForNode(n *m1Node, all []*m1Node) string {
	if n.Level <= 1 || n.Father == "" {
		return n.Name
	}

	// 先构建一个索引：level+name -> 节点
	type key struct {
		Level int
		Name  string
	}
	index := make(map[key]*m1Node)
	for _, x := range all {
		index[key{Level: x.Level, Name: x.Name}] = x
	}

	// 从当前节点往上回溯到 L1
	var chain []*m1Node
	cur := n
	for cur != nil {
		chain = append(chain, cur)
		if cur.Level == 1 || cur.Father == "" {
			break
		}
		parent, ok := index[key{Level: cur.Level - 1, Name: cur.Father}]
		if !ok {
			break
		}
		cur = parent
	}

	// chain 现在是 [当前, 父, 父的父, .]，需要反转
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	names := make([]string, 0, len(chain))
	for _, x := range chain {
		names = append(names, x.Name)
	}
	return strings.Join(names, ".")
}

// 用 txt 文件名（modelName）替换 elementName 的第一段（第一个 '.' 之前）
// - "RUNNABLE"      -> "CL1CM1"
// - "RUNNABLE.DATA" -> "CL1CM1.DATA"
// - "RUNNABLE.A.B"  -> "CL1CM1.A.B"
func replaceElementPrefixWithTxtName(elementName, modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return elementName
	}
	if idx := strings.Index(elementName, "."); idx >= 0 {
		return modelName + elementName[idx:] // 保留 '.' 及后面的部分
	}
	return modelName
}

// 把 nodes 写成一个 ldi.xml 文件
// 注意：只输出 1.maxLevel-1 层的节点，最底层 Level=maxLevel 的节点完全不写入
func writeM1LDI(ldiPath string, modelName string, nodes []*m1Node) error {
	var root ldiRoot

	// 计算全局最大层级
	maxLevel := 0
	for _, n := range nodes {
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	// 为了输出稳定性：按 level 升序，再按层级名排序
	type namedNode struct {
		Node *m1Node
		Path string
	}
	var list []namedNode
	for _, n := range nodes {
		// 跳过最底层：不写入 LDI
		if n.Level >= maxLevel {
			continue
		}
		path := buildHierNameForNode(n, nodes)
		list = append(list, namedNode{Node: n, Path: path})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Node.Level != list[j].Node.Level {
			return list[i].Node.Level < list[j].Node.Level
		}
		return list[i].Path < list[j].Path
	})

	for _, nn := range list {
		n := nn.Node

		// ★核心：写入 ldi.xml 时，用 txt 文件名替换 name 第一段
		name := replaceElementPrefixWithTxtName(nn.Path, modelName)

		el := ldiElement{
			Name: name,
			Property: []ldiProperty{
				{
					Name:  "coverage.m1",
					Value: fmt.Sprintf("%.4f", n.Coverage),
				},
			},
		}
		root.Items = append(root.Items, el)
	}

	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(ldiPath), 0755); err != nil {
		return err
	}

	// 写文件（带 xml 头）
	content := append([]byte(xml.Header), out...)
	return os.WriteFile(ldiPath, content, 0644)
}
