package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
)

func ParseData(data []byte) (map[string]string, map[string]DocPDFData, []DocPDFData) {
	var DocItemList []DocResourceItem
	if err := json.Unmarshal(data, &DocItemList); err != nil {
		return nil, nil, nil
	}

	tagMap := map[string]string{}
	docPDFMap := map[string]DocPDFData{}
	docPDFList := []DocPDFData{}

	for _, item := range DocItemList {
		for _, tag := range item.TagList {
			tagMap[tag.TagID] = tag.TagName
		}

		for _, tagPath := range item.TagPaths {
			parts := strings.Split(tagPath, "/")
			tagID := parts[len(parts)-1]
			tagData := DocPDFData{
				ID:      item.ID,
				Title:   item.Title,
				TagPath: tagPath,
				TagID:   tagID,
			}

			docPDFMap[tagID] = tagData // TODO remove 不唯一
			docPDFList = append(docPDFList, tagData)
		}
	}

	return tagMap, docPDFMap, docPDFList
}

func ParseDataList(dataList [][]byte) (map[string]string, map[string]DocPDFData, []DocPDFData) {
	tagMap := map[string]string{}
	docPDFMap := map[string]DocPDFData{}
	docPDFList := []DocPDFData{}

	for _, data := range dataList {
		partTagMap, partDocPDFMap, partDocPDFList := ParseData(data)

		for k, v := range partTagMap {
			tagMap[k] = v
		}
		for k, v := range partDocPDFMap {
			docPDFMap[k] = v
		}
		docPDFList = append(docPDFList, partDocPDFList...)
	}

	return tagMap, docPDFMap, docPDFList
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

func ReadRawData(name string, local bool) ([]byte, [][]byte) {
	dataDir := "data/tchMaterial"
	tagURL := TchMaterialInfo.Tag
	versionURL := TchMaterialInfo.Version

	var tagData []byte
	dataList := [][]byte{}
	if local {
		// 使用本地文件
		tagURL = path.Join(dataDir, path.Base(tagURL))
		versionURL = path.Join(dataDir, path.Base(versionURL))
	}

	tagData, err := fetchJSONFile(tagURL)
	if err != nil {
		return tagData, dataList
	}

	versionData, err := fetchJSONFile(versionURL)
	if err != nil {
		return tagData, dataList
	}

	urls, err := ParseURLsFromJSON(versionData)
	if err != nil {
		return tagData, dataList
	}

	for _, url := range urls {
		dataURL := url
		if local {
			dataURL = path.Join(dataDir, path.Base(url))
		}
		data, err := fetchJSONFile(dataURL)
		if err != nil {
			continue
		}
		dataList = append(dataList, data)
	}
	return tagData, dataList

}

func FetchRawData(name string, local bool) ([]TagItem, map[string]string, map[string]DocPDFData) {
	var tagItems []TagItem
	tagData, dataList := ReadRawData(name, local)
	if tagData == nil || len(dataList) == 0 {
		return tagItems, nil, nil
	}

	tagBase := ParseHierarchies(tagData)
	tagMap, docPDFMap, _ := ParseDataList(dataList)
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
		if val, ok := docPDFMap[tagItem.TagID]; ok {
			optionNames = append(optionNames, val.Title)
			optionIDs = append(optionIDs, val.ID)
		}
		return tagItem.TagName, optionNames, optionIDs, nil
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
