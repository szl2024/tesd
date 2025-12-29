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

// 1. Windows 경로 읽기: 콘솔에 안내 문구를 출력하고 입력을 받은 뒤, `M1_Public_Data.SrcPath`에 저장합니다.
func ReadWindowsPath() {
	fmt.Print("모델이 저장된 Windows 경로를 입력하세요： ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("입력 읽기 실패:", err)
		return
	}

	input = strings.TrimSpace(input)
	M1_Public_Data.SrcPath = input
}

// 2. SrcPath 하위의 각 하위 폴더에서 동일한 이름의 slx 파일을 찾아 BuildDir로 복사합니다.
//   SrcPath/
//     ├─ ModelA/  →  ModelA/ModelA.slx  →  BuildDir/ModelA.slx로 복사
//     ├─ ModelB/  →  ModelB/ModelB.slx  →  BuildDir/ModelB.slx로 복사
// 또한 TxtDir 하위에 동일한 이름의 txt 파일을 생성합니다: ModelA.txt, ModelB.txt
func CopySlxToBuild() {
	srcRoot := M1_Public_Data.SrcPath
	dstRoot := M1_Public_Data.BuildDir
	txtRoot := M1_Public_Data.TxtDir

	if srcRoot == "" {
		fmt.Println("SrcPath가 비어 있습니다. 먼저 ReadWindowsPath()를 호출하여 경로를 입력하세요.")
		return
	}
	if dstRoot == "" {
		fmt.Println("BuildDir이 비어 있습니다. 먼저 SetWorkDir()를 호출하여 작업 공간을 초기화하세요.")
		return
	}
	if txtRoot == "" {
		fmt.Println("TxtDir이 비어 있습니다. SetWorkDir()가 올바르게 설정되었는지 확인하세요.")
		return
	}

	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		fmt.Println("SrcPath 디렉터리를 읽을 수 없습니다：", err)
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		folderName := e.Name()
		slxPath := filepath.Join(srcRoot, folderName, folderName+".slx")
		if _, err := os.Stat(slxPath); err != nil {
			// 동일한 이름의 slx 파일이 없으면 건너뜁니다.
			continue
		}

		// 대상 slx 파일 경로: BuildDir/동일한이름.slx
		dstPath := filepath.Join(dstRoot, folderName+".slx")

		// slx 파일 복사
		if err := copyFile(slxPath, dstPath); err != nil {
			fmt.Printf("复制失败 [%s] → [%s]：%v\n", slxPath, dstPath, err)
			continue
		}

		// TxtDir 아래에 동일한 이름의 txt 파일을 생성합니다.
		txtPath := filepath.Join(txtRoot, folderName+".txt")
		f, err := os.Create(txtPath) // 실행할 때마다 재생성/초기화합니다.
		if err != nil {
			fmt.Printf("无法创建 txt 文件 [%s]：%v\n", txtPath, err)
			continue
		}
		_ = f.Close()
	}
}

// 4. BuildDir 아래의 slx 파일을 동일한 이름의 디렉터리로 압축 해제합니다.
//   BuildDir/
//     ├─ ModelA.slx  →  BuildDir/ModelA/...에 압축 해제
//     ├─ ModelB.slx  →  BuildDir/ModelB/...에 압축 해제
func UnzipSlxFiles() {
	buildRoot := M1_Public_Data.BuildDir
	if buildRoot == "" {
		fmt.Println("BuildDir이 비어 있습니다. 먼저 SetWorkDir()를 호출하여 작업 공간을 초기화하세요.")
		return
	}

	entries, err := os.ReadDir(buildRoot)
	if err != nil {
		fmt.Println("BuildDir 디렉터리를 읽을 수 없습니다:", err)
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

		// 압축 해제 대상 디렉터리가 깨끗한 상태(기존 파일 없음)인지 보장합니다.
		_ = os.RemoveAll(destDir)

		if err := unzipOne(slxPath, destDir); err != nil {
			fmt.Printf("압축 해제 실패 [%s] → [%s]：%v\n", slxPath, destDir, err)
			continue
		}
	}
}

// 간단한 파일 복사 유틸리티
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// 대상 디렉터리가 존재하도록 보장합니다.
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

// 단일 slx(zip) 파일을 destDir에 압축 해제합니다.
func unzipOne(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)

		// 디렉터리
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		// 상위 디렉터리가 존재하도록 보장합니다.
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

// ===================== M1 LDI 생성 관련 =====================

// 用于从 txt 中解析出来的节点信息
type m1Node struct {
	Level          int
	Name           string
	SID            string
	Father         string
	Ports          int     // 현재 노드의 포트 개수(virtual port 포함)
	CSPorts        int     // L1의 C-S 포트 개수(해당 레벨에만 적용)
	ChildCount     int     // 직접 하위 노드 개수
	ChildPorts     int     // 직접 하위 노드들의 포트 수 합계
	EffectivePorts float64 // L1: 가중 포트 수, 기타 레벨: Ports와 동일
	Coverage       float64 // 계산된 m1 값
}

// LDI XML 구조
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

// 6. TxtDir 하위의 txt 파일을 기반으로 해당 ldi.xml을 생성합니다.
//    예: TurnLight.txt → TurnLight.ldi.xml
//    규칙: N단계가 존재할 경우 1..N-1 단계까지만 m1을 계산하고 출력하며, 최하위 N단계는 출력하지 않습니다.
//    또한 TxtDir 하위에 XXX_m1.txt를 생성하여 각 레벨별 Ports / 하위 노드 개수 / 하위 포트 수를 요약합니다.
func GenerateM1LDIFromTxt() {
	txtRoot := M1_Public_Data.TxtDir
	ldiRoot := M1_Public_Data.LDIDir

	if txtRoot == "" || ldiRoot == "" {
		fmt.Println("TxtDir 또는 LDIDir이 비어 있습니다. SetWorkDir()가 올바르게 설정되었는지 확인하세요.")
		return
	}

	entries, err := os.ReadDir(txtRoot)
	if err != nil {
		fmt.Println("TxtDir 읽기 실패:", err)
		return
	}

	// LDI 디렉터리가 존재하도록 보장합니다.
	if err := os.MkdirAll(ldiRoot, 0755); err != nil {
		fmt.Println("LDI 디렉터리 생성 실패:", err)
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
			fmt.Printf("txt 파싱 실패 [%s]: %v\n", txtPath, err)
			continue
		}
		if len(nodes) == 0 {
			fmt.Printf("txt에서 노드를 파싱하지 못했습니다. [%s]\n", txtPath)
			continue
		}

		computeM1ForNodes(nodes)

		// ldi.xml을 생성합니다(여기서 txt 파일명을 전달하여 element name의 접두어를 치환하는 데 사용합니다).
		ldiPath := filepath.Join(ldiRoot, modelName+".ldi.xml")
		if err := writeM1LDI(ldiPath, modelName, nodes); err != nil {
			fmt.Printf("LDI 작성 실패 [%s]: %v\n", ldiPath, err)
			// 중단하지 않고, 계속해서 m1.txt를 생성합니다.
		} else {
			fmt.Printf("📄 M1 지표 계산 완료: %s\n", ldiPath)
		}

		// XXX_m1.txt 생성
		statsPath := filepath.Join(txtRoot, modelName+"_m1.txt")
		if err := writeM1StatsTxt(statsPath, nodes); err != nil {
			fmt.Printf("m1 통계 작성 실패 [%s]: %v\n", statsPath, err)
		} else {
			//fmt.Printf("m1 통계 생성 완료：%s\n", statsPath)
		}
	}
}

// 하나의 txt를 파싱하여 모든 [Lx] block과 [Lx Port]/[Lx virtual Port]를 모두 추출합니다.
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

		// Block 행: "[L"로 시작하고 앞에 탭(Tab) 들여쓰기가 없는 행
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

		// 포트 행 예: \t[L1 Port] 또는 \t[L2 virtual Port]
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

			// 현재 노드의 동일 레벨 포트만 카운트합니다.
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

// [L2 Port] 또는 [L2 virtual Port] 같은 문자열에서 Level을 파싱합니다.
// 또한 한 줄 전체에서 PortType을 파싱합니다( C-S 포트 식별 용도로만 사용 ).
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

// 다음과 같은 형식을 파싱합니다:
// [L2] Name: HazardCtrlLogic	BlockType=SubSystem	SID=66       	FatherNode=TurnLight_Runnable_10ms_sys
func parseBlockLineInfo(trim string) (int, string, string, string, bool) {
	// 레벨(층) 파싱
	levelRe := regexp.MustCompile(`^\[L(\d+)\]`)
	m := levelRe.FindStringSubmatch(trim)
	if len(m) < 2 {
		return 0, "", "", "", false
	}
	level, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", "", "", false
	}

	// Name은 Name:와 BlockType= 사이에 있습니다.
	name := ""
	if nameIdx := strings.Index(trim, "Name:"); nameIdx >= 0 {
		after := trim[nameIdx+len("Name:"):]
		btIdx := strings.Index(after, "BlockType=")
		if btIdx > 0 {
			name = strings.TrimSpace(after[:btIdx])
		} else {
			// BlockType=가 없으면 행 끝까지를 사용합니다.
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

// 규칙에 따라 각 노드의 m1을 계산합니다.
// - N단계가 있을 때 1..N-1 단계만 계산하며, 마지막 단계(Level=N) 노드의 coverage는 0입니다.
// - 또한 ChildCount / ChildPorts / EffectivePorts를 함께 채워 ldi와 _m1.txt에서 공통으로 사용합니다.
func computeM1ForNodes(nodes []*m1Node) {
	if len(nodes) == 0 {
		return
	}

	// 1) 최대 레벨을 찾고, 각 노드의 EffectivePorts를 미리 계산합니다.
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

	// 2) 레벨별로 그룹화하여 하위 노드를 쉽게 조회할 수 있도록 합니다.
	levelMap := make(map[int][]*m1Node)
	for _, n := range nodes {
		levelMap[n.Level] = append(levelMap[n.Level], n)
	}

	// 3) 각 노드별로 m1과 하위 노드 통계를 순차적으로 계산합니다.
	for _, n := range nodes {
		// 초기화
		n.ChildCount = 0
		n.ChildPorts = 0
		n.Coverage = 0

		// 최하위 레벨(다음 레벨이 없거나 전체 최대 레벨인 경우)은 coverage=0으로 설정하고 더 이상 계산하지 않습니다.
		if n.Level >= maxLevel {
			continue
		}

		childLevel := n.Level + 1
		children := levelMap[childLevel]

		// 실제 ‘직접 하위 노드’만 필터링합니다: FatherName == 현재 노드 Name
		var realChildren []*m1Node
		for _, c := range children {
			if c.Father == n.Name {
				realChildren = append(realChildren, c)
			}
		}

		// 하위 노드 포트 수의 합계
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

		// L1 노드만 C-S 포트에 대해 1.2 가중치를 적용합니다.
		if n.Level == 1 {
			n.Coverage = n.EffectivePorts * float64(n.ChildCount) * float64(n.ChildPorts)
		} else {
			// L2 이후(마지막에서 두 번째 레벨까지)는 단순 개수 기반으로 계산합니다.
			n.Coverage = float64(n.Ports) * float64(n.ChildCount) * float64(n.ChildPorts)
		}
	}
}

// 레벨(계층) 이름 구성：
// L1: Name
// L2: Father.Name  => L1.Name + "." + L2.Name
// L3: L1.Name + "." + L2.Name + "." + L3.Name
func buildHierNameForNode(n *m1Node, all []*m1Node) string {
	if n.Level <= 1 || n.Father == "" {
		return n.Name
	}

	// 먼저 인덱스를 구성합니다: level+name → 노드
	type key struct {
		Level int
		Name  string
	}
	index := make(map[key]*m1Node)
	for _, x := range all {
		index[key{Level: x.Level, Name: x.Name}] = x
	}

	// 현재 노드에서 위로 거슬러 올라가 L1까지 추적합니다.
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

	// 현재 chain은 [현재, 부모, 조부모, ...] 순서이므로, 반전(역순)해야 합니다.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	names := make([]string, 0, len(chain))
	for _, x := range chain {
		names = append(names, x.Name)
	}
	return strings.Join(names, ".")
}

// element name의 첫 번째 구간(세그먼트)을 txt 파일명(modelName)으로 치환합니다.
// - "RUNNABLE" -> "CL1CM1"
// - "RUNNABLE.DATA" -> "CL1CM1.DATA"
// - "RUNNABLE.DATA.X" -> "CL1CM1.DATA.X"
func replaceElementPrefixWithTxtName(elementName, modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return elementName
	}
	if idx := strings.Index(elementName, "."); idx >= 0 {
		return modelName + elementName[idx:]
	}
	return modelName
}

// nodes를 하나의 ldi.xml 파일로 작성합니다.
// 주의: 1..(maxLevel-1) 레벨의 노드만 출력하며, 최하위 레벨(Level=maxLevel) 노드는 아예 작성하지 않습니다.
func writeM1LDI(ldiPath string, modelName string, nodes []*m1Node) error {
	var root ldiRoot

	// 전체 최대 레벨을 계산합니다.
	maxLevel := 0
	for _, n := range nodes {
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	// 출력 안정성을 위해 level 오름차순으로 정렬한 뒤, 레벨 이름(계층명) 기준으로 다시 정렬합니다.
	type namedNode struct {
		Node *m1Node
		Path string
	}
	var list []namedNode
	for _, n := range nodes {
		// 최하위 레벨은 건너뜁니다: LDI에 작성하지 않습니다.
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
		// ✅ ldi.xml 생성 시 name의 첫 번째 구간을 txt 파일명으로 치환합니다.
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
		return fmt.Errorf("LDI XML 직렬화 실패: %v", err)
	}

	content := append([]byte(xml.Header), out...)
	if err := os.WriteFile(ldiPath, content, 0644); err != nil {
		return fmt.Errorf("LDI 파일 쓰기 실패: %v", err)
	}
	return nil
}

// XXX_m1.txt를 생성하여 각 레벨 노드별로 ‘자체 포트 수’, ‘하위 노드 개수’, ‘하위 노드 포트 총합’을 요약합니다.
// maxLevel-1 레벨까지만 출력합니다.
func writeM1StatsTxt(statsPath string, nodes []*m1Node) error {
	if len(nodes) == 0 {
		return nil
	}

	// 전체 최대 레벨을 계산합니다.
	maxLevel := 0
	for _, n := range nodes {
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	// 정렬: 레벨, 이름 기준
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
		// N-1층까지만 출력하고, 최하층은 출력하지 않습니다.
		if n.Level >= maxLevel {
			continue
		}
		lv := n.Level

		// L1: 포트 수에 C-S 가중치 적용
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
			// L2 및 이후: 포트는 가중치를 적용하지 않고, Ports를 그대로 사용합니다.
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
