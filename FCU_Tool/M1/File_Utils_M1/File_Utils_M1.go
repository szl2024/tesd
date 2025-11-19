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
//     ├─ ModelA.slx  → 解压到 BuildDir/ModelA/...
//     ├─ ModelB.slx  → 解压到 BuildDir/ModelB/...
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
	XMLName xml.Name    `xml:"ldi"`
	Items   []ldiElement `xml:"element"`
}

// 6. 根据 TxtDir 下的 txt 文件生成对应的 ldi.xml
//    例如 TurnLight.txt -> TurnLight.ldi.xml
//    规则：如果存在 N 层，只对 1..N-1 层计算并输出 m1，最底层 N 不输出
//    同时在 TxtDir 下生成 XXX_m1.txt，总结每层的 Ports / 子节点个数 / 子端口数
func GenerateM1LDIFromTxt() {
	txtRoot := M1_Public_Data.TxtDir
	ldiRoot := M1_Public_Data.LDIDir

	if txtRoot == "" || ldiRoot == "" {
		fmt.Println("TxtDir 或 LDIDir 为空，请检查 SetWorkDir 是否正确设置")
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
		if err := writeM1LDI(ldiPath, nodes); err != nil {
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
			trim := strings.TrimLeft(line, "\t")
			endIdx := strings.Index(trim, "]")
			if endIdx <= 0 {
				continue
			}
			header := trim[1:endIdx] // e.g. "L1 Port" or "L2 virtual Port"

			level, portType, ok := parsePortLineLevelAndType(header, trim)
			if !ok {
				continue
			}

			// 只在当前节点的同层端口上计数
			if curNode != nil && curNode.Level == level {
				curNode.Ports++
				if portType == "C-S" {
					curNode.CSPorts++
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

// 从类似 "[L2 Port]" 或 "[L2 virtual Port]" 里解析出 Level
// 同时从整行里解析 PortType（只用来识别 C-S port）
func parsePortLineLevelAndType(header string, fullLine string) (int, string, bool) {
	fields := strings.Fields(header) // e.g. ["L2","Port"] or ["L2","virtual","Port"]
	if len(fields) == 0 {
		return 0, "", false
	}
	levelStr := strings.TrimPrefix(fields[0], "L")
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return 0, "", false
	}

	portType := ""
	if idx := strings.Index(fullLine, "PortType="); idx >= 0 {
		rest := fullLine[idx+len("PortType="):]
		ptFields := strings.Fields(rest)
		if len(ptFields) > 0 {
			portType = strings.TrimSpace(ptFields[0])
		}
	}
	return level, portType, true
}

// 解析类似：
// [L2] Name: HazardCtrlLogic	BlockType=SubSystem	SID=66       	FatherNode=TurnLight_Runnable_10ms_sys
func parseBlockLineInfo(trim string) (int, string, string, string, bool) {
	// 解析层级
	levelRe := regexp.MustCompile(`^\[L(\d+)\]`)
	m := levelRe.FindStringSubmatch(trim)
	if len(m) < 2 {
		return 0, "", "", "", false
	}
	level, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", "", "", false
	}

	// Name 在 "Name:" 和 "BlockType=" 之间
	name := ""
	if nameIdx := strings.Index(trim, "Name:"); nameIdx >= 0 {
		after := trim[nameIdx+len("Name:"):]
		btIdx := strings.Index(after, "BlockType=")
		if btIdx > 0 {
			name = strings.TrimSpace(after[:btIdx])
		} else {
			// 没有 BlockType= 时，取到行尾
			name = strings.TrimSpace(after)
		}
	}

	// SID=
	sid := ""
	if sidIdx := strings.Index(trim, "SID="); sidIdx >= 0 {
		after := trim[sidIdx+len("SID="):]
		sidFields := strings.Fields(after)
		if len(sidFields) > 0 {
			sid = sidFields[0]
		}
	}

	// FatherNode=
	father := ""
	if faIdx := strings.Index(trim, "FatherNode="); faIdx >= 0 {
		after := trim[faIdx+len("FatherNode="):]
		faFields := strings.Fields(after)
		if len(faFields) > 0 {
			father = faFields[0]
		}
	}

	if name == "" {
		return 0, "", "", "", false
	}
	return level, name, sid, father, true
}

// 按你的规则计算每个节点的 m1
// - 有 N 层，只对 1..N-1 层计算（最后一层 Level=N 的节点 coverage=0）
// - 同时填充 ChildCount / ChildPorts / EffectivePorts，供 ldi 和 _m1.txt 共用
func computeM1ForNodes(nodes []*m1Node) {
	if len(nodes) == 0 {
		return
	}

	// 1) 找出最大层级，同时预先计算每个节点的 EffectivePorts
	maxLevel := 0
	for _, n := range nodes {
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
		if n.Level == 1 {
			normalPorts := n.Ports - n.CSPorts
			if normalPorts < 0 {
				normalPorts = 0
			}
			n.EffectivePorts = float64(normalPorts) + float64(n.CSPorts)*1.2
		} else {
			n.EffectivePorts = float64(n.Ports)
		}
	}

	// 2) 按层级分组，方便查找子节点
	levelMap := make(map[int][]*m1Node)
	for _, n := range nodes {
		levelMap[n.Level] = append(levelMap[n.Level], n)
	}

	// 3) 逐个节点计算 m1 和子节点统计
	for _, n := range nodes {
		// 默认初始化
		n.ChildCount = 0
		n.ChildPorts = 0
		n.Coverage = 0

		// 最深层（没有下一层）或已经是全局最大层级：coverage=0，不再计算
		if n.Level >= maxLevel {
			continue
		}

		childLevel := n.Level + 1
		children := levelMap[childLevel]

		// 筛选真正的“直属子节点”：FatherName == 当前节点 Name
		var realChildren []*m1Node
		for _, c := range children {
			if c.Father == n.Name {
				realChildren = append(realChildren, c)
			}
		}

		// 子节点端口数之和
		pChildSum := 0
		for _, c := range realChildren {
			pChildSum += c.Ports
		}

		n.ChildCount = len(realChildren)
		n.ChildPorts = pChildSum

		if n.ChildCount == 0 || n.ChildPorts == 0 {
			n.Coverage = 0
			continue
		}

		// 仅 L1 节点按 C-S 端口做 1.2 计权
		if n.Level == 1 {
			n.Coverage = n.EffectivePorts * float64(n.ChildCount) * float64(n.ChildPorts)
		} else {
			// L2 及之后（直到倒数第二层）：纯粹数量算法
			n.Coverage = float64(n.Ports) * float64(n.ChildCount) * float64(n.ChildPorts)
		}
	}
}

// 构造层级名字：
// L1: Name
// L2: Father.Name  => L1.Name + "." + L2.Name
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

	// chain 现在是 [当前, 父, 父的父, ...]，需要反转
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	names := make([]string, 0, len(chain))
	for _, x := range chain {
		names = append(names, x.Name)
	}
	return strings.Join(names, ".")
}

// 把 nodes 写成一个 ldi.xml 文件
// 注意：只输出 1..maxLevel-1 层的节点，最底层 Level=maxLevel 的节点完全不写入
func writeM1LDI(ldiPath string, nodes []*m1Node) error {
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
		name := nn.Path

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
		return fmt.Errorf("序列化 LDI XML 失败: %v", err)
	}

	content := append([]byte(xml.Header), out...)
	if err := os.WriteFile(ldiPath, content, 0644); err != nil {
		return fmt.Errorf("写入 LDI 文件失败: %v", err)
	}
	return nil
}

// 生成 XXX_m1.txt，总结每个层级节点的：自身端口数、子节点个数、子节点端口总数
// 仅输出到 maxLevel-1 层
func writeM1StatsTxt(statsPath string, nodes []*m1Node) error {
	if len(nodes) == 0 {
		return nil
	}

	// 计算全局最大层级
	maxLevel := 0
	for _, n := range nodes {
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	// 排序：按层级、名字
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Level != nodes[j].Level {
			return nodes[i].Level < nodes[j].Level
		}
		return nodes[i].Name < nodes[j].Name
	})

	f, err := os.Create(statsPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, n := range nodes {
		// 只输出到 N-1 层，最底层不输出
		if n.Level >= maxLevel {
			continue
		}
		lv := n.Level

		// L1: 端口数带 C-S 权重
		if lv == 1 {
			line := fmt.Sprintf(
				"[L1] Name: %s\tL1Ports(Weighted)=%.1f\tL2Count=%d\tL2Ports=%d\n",
				n.Name,
				n.EffectivePorts,
				n.ChildCount,
				n.ChildPorts,
			)
			if _, err := f.WriteString(line); err != nil {
				return err
			}
		} else {
			// L2 及之后：端口不加权，直接用 Ports
			nextLevel := lv + 1
			line := fmt.Sprintf(
				"[L%d] Name: %s\tL%dPorts=%d\tL%dCount=%d\tL%dPorts=%d\n",
				lv,
				n.Name,
				lv, n.Ports,
				nextLevel, n.ChildCount,
				nextLevel, n.ChildPorts,
			)
			if _, err := f.WriteString(line); err != nil {
				return err
			}
		}
	}

	return nil
}
