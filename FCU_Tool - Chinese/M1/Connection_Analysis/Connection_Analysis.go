package Connection_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// P 标签
type xmlP struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

// Branch 标签
type xmlBranch struct {
	Ps []xmlP `xml:"P"`
}

// Line 标签
type xmlLine struct {
	Ps       []xmlP     `xml:"P"`
	Branches []xmlBranch `xml:"Branch"`
}

// 只关心 Line 的 System
type xmlSystem struct {
	Lines []xmlLine `xml:"Line"`
}

// 一条连接边：SrcSID → DstSID
type Edge struct {
	SrcSID string
	DstSID string
}

// 解析某个 system_xxx.xml 中的所有连接，返回 Edge 列表
func AnalyzeConnectionsInFile(dir, file string) ([]Edge, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 XML 失败 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("解析 XML 失败 [%s]: %w", fullPath, err)
	}

	var edges []Edge

	for _, line := range sys.Lines {
		var srcSID string

		// 找这一条 Line 的 Src
		for _, p := range line.Ps {
			if p.Name == "Src" {
				srcSID = parseSIDFromEndpoint(p.Value)
				break
			}
		}
		if srcSID == "" {
			continue
		}

		// 1）主 Line 上可能有一个 Dst
		for _, p := range line.Ps {
			if p.Name == "Dst" {
				dstSID := parseSIDFromEndpoint(p.Value)
				if dstSID != "" {
					edges = append(edges, Edge{
						SrcSID: srcSID,
						DstSID: dstSID,
					})
				}
			}
		}

		// 2）每个 Branch 里也可能有 Dst
		for _, br := range line.Branches {
			for _, p := range br.Ps {
				if p.Name == "Dst" {
					dstSID := parseSIDFromEndpoint(p.Value)
					if dstSID != "" {
						edges = append(edges, Edge{
							SrcSID: srcSID,
							DstSID: dstSID,
						})
					}
				}
			}
		}
	}

	return edges, nil
}

// "39#out:1" / "66#in:3" / "202#trigger" → "39" / "66" / "202"
func parseSIDFromEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ""
	}
	if idx := strings.Index(ep, "#"); idx > 0 {
		return ep[:idx]
	}
	return ep
}
