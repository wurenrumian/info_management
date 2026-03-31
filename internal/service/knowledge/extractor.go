package knowledge

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ExtractTextFromFile extracts searchable plain text from a document file.
func ExtractTextFromFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		text, err := extractDocx(path)
		if err != nil {
			return ""
		}
		return text
	case ".xlsx":
		text, err := extractXlsx(path)
		if err != nil {
			return ""
		}
		return text
	case ".pdf":
		// PDF正文抽取后续可接入专用解析库；当前先存附件元信息。
		return ""
	default:
		return ""
	}
}

func extractDocx(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	xmlData, err := readZipFile(zr.File, "word/document.xml")
	if err != nil || len(xmlData) == 0 {
		return "", err
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	parts := make([]string, 0, 64)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		ch, ok := tok.(xml.CharData)
		if !ok {
			continue
		}
		t := strings.TrimSpace(string(ch))
		if t != "" {
			parts = append(parts, t)
		}
	}
	return normalizeText(strings.Join(parts, " ")), nil
}

func extractXlsx(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	sharedMap, err := readSharedStrings(zr.File)
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, 64)
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		data, err := readZipFile(zr.File, f.Name)
		if err != nil {
			continue
		}
		txt := extractWorksheetText(data, sharedMap)
		if txt != "" {
			parts = append(parts, txt)
		}
	}
	return normalizeText(strings.Join(parts, " ")), nil
}

type sst struct {
	SI []struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func readSharedStrings(files []*zip.File) (map[int]string, error) {
	data, err := readZipFile(files, "xl/sharedStrings.xml")
	if err != nil || len(data) == 0 {
		return map[int]string{}, nil
	}
	var table sst
	if err := xml.Unmarshal(data, &table); err != nil {
		return map[int]string{}, err
	}
	out := make(map[int]string, len(table.SI))
	for i, si := range table.SI {
		if strings.TrimSpace(si.T) != "" {
			out[i] = si.T
			continue
		}
		var b strings.Builder
		for _, r := range si.R {
			b.WriteString(r.T)
		}
		out[i] = b.String()
	}
	return out, nil
}

func extractWorksheetText(data []byte, shared map[int]string) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	parts := make([]string, 0, 64)

	inCell := false
	cellType := ""
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "c" {
				inCell = true
				cellType = ""
				for _, a := range se.Attr {
					if a.Name.Local == "t" {
						cellType = a.Value
					}
				}
			}
			if inCell && se.Name.Local == "v" {
				var v string
				if err := decoder.DecodeElement(&v, &se); err != nil {
					continue
				}
				v = strings.TrimSpace(v)
				if v == "" {
					continue
				}
				if cellType == "s" {
					idx, err := strconv.Atoi(v)
					if err != nil {
						continue
					}
					if s, ok := shared[idx]; ok && strings.TrimSpace(s) != "" {
						parts = append(parts, s)
					}
				} else {
					parts = append(parts, v)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "c" {
				inCell = false
				cellType = ""
			}
		}
	}
	return normalizeText(strings.Join(parts, " "))
}

func readZipFile(files []*zip.File, name string) ([]byte, error) {
	for _, f := range files {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, nil
}

var multiSpace = regexp.MustCompile(`\s+`)

func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return multiSpace.ReplaceAllString(s, " ")
}
