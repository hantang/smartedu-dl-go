package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
)

func ParseData(data []byte) (map[string]string, map[string]DocPDFData) {
	var DocItemList []DocResourceItem
	if err := json.Unmarshal(data, &DocItemList); err != nil {
		return nil, nil
	}

	tagMap := map[string]string{}
	docPDFMap := map[string]DocPDFData{}

	for _, item := range DocItemList {
		for _, tag := range item.TagList {
			tagMap[tag.TagID] = tag.TagName
		}

		for _, tagPath := range item.TagPaths {
			parts := strings.Split(tagPath, "/")
			tagID := parts[len(parts)-1]

			docPDFMap[tagID] = DocPDFData{
				ID:      item.ID,
				Title:   item.Title,
				TagPath: tagPath,
			}
		}
	}

	return tagMap, docPDFMap
}

func ParseDataList(dataList [][]byte) (map[string]string, map[string]DocPDFData) {
	tagMap := map[string]string{}
	docPDFMap := map[string]DocPDFData{}
	for _, data := range dataList {
		partTagMap, partDocPDFMap := ParseData(data)

		for k, v := range partTagMap {
			tagMap[k] = v
		}
		for k, v := range partDocPDFMap {
			docPDFMap[k] = v
		}
	}
	return tagMap, docPDFMap
}

func ParseHierarchies(data []byte) TagBase {
	// 解析 tch_material_tag.json
	var tagBase TagBase
	if err := json.Unmarshal(data, &tagBase); err != nil {
		fmt.Println("Error unmarshaling:", err)
	}
	return tagBase
}

func fetchJSONFile(filename string) ([]byte, error) {
	slog.Debug("process filename = " + filename)
	if strings.HasPrefix(filename, "http") {
		return FetchJsonData(filename)
	}

	data, err := os.ReadFile(filename)
	return data, err
}

func ParseURLsFromJSON(data []byte) ([]string, error) {
	// 抽取urls字段
	var dv DataVersion
	if err := json.Unmarshal(data, &dv); err != nil {
		return nil, err
	}

	switch v := dv.URLs.(type) {
	case string:
		return strings.Split(v, ","), nil
	case []interface{}:
		urls := make([]string, len(v))
		for i, url := range v {
			if s, ok := url.(string); ok {
				urls[i] = s
			}
		}
		return urls, nil
	default:
		return nil, nil
	}
}

func FetchRawData(name string, local bool) ([]TagItem, map[string]string, map[string]DocPDFData) {
	dataDir := "data/tchMaterial"
	tagURL := TchMaterialInfo.Tag
	versionURL := TchMaterialInfo.Version
	tagFile := tagURL
	versionFile := versionURL
	if local {
		tagFile = path.Join(dataDir, path.Base(tagURL))
		versionFile = path.Join(dataDir, path.Base(versionURL))
	}

	var tagItems []TagItem
	tagData, err := fetchJSONFile(tagFile)
	if err != nil {
		return tagItems, nil, nil
	}

	versionData, err := fetchJSONFile(versionFile)
	if err != nil {
		return tagItems, nil, nil
	}

	urls, err := ParseURLsFromJSON(versionData)
	if err != nil {
		return tagItems, nil, nil
	}

	dataList := [][]byte{}
	for _, url := range urls {
		tmpFile := url
		if local {
			tmpFile = path.Join(dataDir, path.Base(url))
		}
		data, err := fetchJSONFile(tmpFile)
		if err != nil {
			continue
		}
		dataList = append(dataList, data)
	}

	tagMap, docPDFMap := ParseDataList(dataList)
	tagBase := ParseHierarchies(tagData)
	if len(tagBase.Hierarchies) > 0 && len(tagBase.Hierarchies[0].Children) > 0 {
		tagItems = tagBase.Hierarchies[0].Children
	}

	return tagItems, tagMap, docPDFMap
}

func Query(tagItem TagItem, docPDFMap map[string]DocPDFData) (string, []string, []string, []TagItem) {
	optionIDs := []string{}
	optionNames := []string{}

	hierarchies := tagItem.Hierarchies
	if hierarchies == nil {
		return "", optionNames, optionIDs, nil
	}

	hierarchy := hierarchies[0]
	title := hierarchy.HierarchyName
	children := hierarchy.Children

	if len(children) > 0 {
		for _, child := range children {
			// slog.Debug(fmt.Sprintf("child name = %s, id = %s", child.TagName, child.TagID))
			optionNames = append(optionNames, child.TagName)
			optionIDs = append(optionIDs, child.TagID)
		}
		// optionIDs = hierarchy.Ext.HasNextTagPath
	} else {
		for _, hidden := range hierarchy.Ext.HiddenTags {
			if val, ok := docPDFMap[hidden]; ok {
				optionNames = append(optionNames, val.Title)
				optionIDs = append(optionIDs, val.ID)
			}
		}
	}

	// slog.Debug("Title: " + title)
	// slog.Debug("==> OptionNames: " + strings.Join(optionNames, ", "))
	// slog.Info("optionIDs: " + strings.Join(optionIDs, ", "))
	return title, optionNames, optionIDs, children
}

func GenerateURLFromID(bookIdList []string) []string {
	// book_id转化成URL
	// name := "/tchMaterial/detail"
	example_url := TchMaterialInfo.detail
	urls := []string{}
	for _, book_id := range bookIdList {
		urls = append(urls, fmt.Sprintf(example_url, book_id))
	}
	return urls
}
