package Connection_Analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// P 태그
type xmlP struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

// Branch 태그
type xmlBranch struct {
	Ps []xmlP `xml:"P"`
}

// Line 태그
type xmlLine struct {
	Ps       []xmlP     `xml:"P"`
	Branches []xmlBranch `xml:"Branch"`
}

// Line에 포함된 System만 분석 대상으로 합니다
type xmlSystem struct {
	Lines []xmlLine `xml:"Line"`
}

// 하나의 연결 엣지: SrcSID → DstSID
type Edge struct {
	SrcSID string
	DstSID string
}

// 특정 system_xxx.xml의 모든 연결을 파싱하여 Edge 리스트를 반환합니다.
func AnalyzeConnectionsInFile(dir, file string) ([]Edge, error) {
	fullPath := filepath.Join(dir, file)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("XML 읽기 실패 [%s]: %w", fullPath, err)
	}

	var sys xmlSystem
	if err := xml.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("XML 파싱 실패 [%s]: %w", fullPath, err)
	}

	var edges []Edge

	for _, line := range sys.Lines {
		var srcSID string

		// 해당 Line의 Src를 찾습니다.
		for _, p := range line.Ps {
			if p.Name == "Src" {
				srcSID = parseSIDFromEndpoint(p.Value)
				break
			}
		}
		if srcSID == "" {
			continue
		}

		// 1）메인 Line에는 Dst가 하나 있을 수 있습니다.
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

		// 2）각 Branch에도 Dst가 있을 수 있습니다.
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
