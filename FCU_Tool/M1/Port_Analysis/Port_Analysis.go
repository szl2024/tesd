package Port_Analysis

import (
	"encoding/xml"
	"regexp"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// ParsePortBlock 解析 <Block BlockType="Inport|Outport" ...>，返回 PortInfo；
// 若 se 不是端口块，返回 ok=false；若是端口块，消费完整元素并返回 ok=true。
func ParsePortBlock(dec *xml.Decoder, se xml.StartElement) (p M1_Public_Data.PortInfo, ok bool, err error) {
	if !strings.EqualFold(se.Name.Local, "Block") {
		return p, false, nil
	}

	var blockType, name, sid string
	for _, a := range se.Attr {
		switch strings.ToLower(a.Name.Local) {
		case "blocktype":
			blockType = a.Value
		case "name":
			name = a.Value
		case "sid":
			sid = a.Value
		}
	}
	isIn := strings.EqualFold(blockType, "Inport")
	isOut := strings.EqualFold(blockType, "Outport")
	if !isIn && !isOut {
		// 非端口块：不消费，由上层处理
		return p, false, nil
	}

	// 消费完整 <Block> ... </Block>
	if err = skipElement(dec, se.Name); err != nil {
		return p, true, err
	}

	if name == "" || sid == "" {
		return p, true, nil // 字段不完整则忽略
	}

	// 规范化 SID：
	// 1) 兼容形如 "69::102"（前者为所属 SubSystem 的 SID，后者为端口自身 SID）
	// 2) 取最后一段的数字作为端口 SID，方便与 <Line><P Name="Src">102#out:1</P> 做映射
	sid = normalizeSID(sid)

	pt := M1_Public_Data.PortIn
	if isOut {
		pt = M1_Public_Data.PortOut
	}
	p = M1_Public_Data.PortInfo{
		Name: name,
		SID:  sid,
		Type: pt,
	}
	return p, true, nil
}

// normalizeSID:
// - 如果 SID 中包含 "::"，取最后一段
// - 仅保留其中的数字（有些工具链可能附带额外字符）
func normalizeSID(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "::"); i >= 0 && i+2 < len(s) {
		s = s[i+2:]
	}
	// 只保留数字
	digits := digitRe.FindString(s)
	if digits != "" {
		return digits
	}
	return s
}

var digitRe = regexp.MustCompile(`[0-9]+`)

// 本地 skip：跳过 name 所代表元素的整棵子树
func skipElement(dec *xml.Decoder, name xml.Name) error {
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return nil
}
// ParsePortsInXML 解析整个 system_root.xml 内容中的所有 Inport / Outport Block
func ParsePortsInXML(xmlData string) ([]*M1_Public_Data.PortInfo, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlData))
	var ports []*M1_Public_Data.PortInfo

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return ports, err
		}

		switch se := tok.(type) {
		case xml.StartElement:
			if strings.EqualFold(se.Name.Local, "Block") {
				var blockType, name, sid string
				for _, a := range se.Attr {
					switch strings.ToLower(a.Name.Local) {
					case "blocktype":
						blockType = a.Value
					case "name":
						name = a.Value
					case "sid":
						sid = a.Value
					}
				}

				if strings.EqualFold(blockType, "Inport") || strings.EqualFold(blockType, "Outport") {
					ioType := "IN"
					if strings.EqualFold(blockType, "Outport") {
						ioType = "OUT"
					}

					port := &M1_Public_Data.PortInfo{
						Name: name,
						SID:  sid,
						Type: "S-R", // 固定为 S-R
						IO:   ioType,
					}
					ports = append(ports, port)
				}
			}
		}
	}

	return ports, nil
}
