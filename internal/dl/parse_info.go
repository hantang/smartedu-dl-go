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

func fetchJSONFile(filename string) ([]byte, error, bool) {
	slog.Debug("process filename = " + filename)
	if strings.HasPrefix(filename, "http") {
		return FetchJsonData(filename)
	}

	data, err := os.ReadFile(filename)
	return data, err, true
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

	tagData, err, statusOK := fetchJSONFile(tagURL)
	if err != nil && statusOK {
		return tagData, dataList
	}

	versionData, err, statusOK := fetchJSONFile(versionURL)
	if err != nil && statusOK {
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
		data, err, statusOK := fetchJSONFile(dataURL)
		if err != nil && statusOK {
			continue
		}
		dataList = append(dataList, data)
	}
	return tagData, dataList

}

func GenerateURLFromID(bookIdList []string) []string {
	// book_id转化成URL
	example_url := TchMaterialInfo.Detail
	urls := []string{}
	for _, book_id := range bookIdList {
		urls = append(urls, fmt.Sprintf(example_url, book_id))
	}
	return urls
}

func ParseHierarchies2(level int, tagItem TagItem, tagMap map[string]string) BookItem {
	var bookItem BookItem
	hierarchies := tagItem.Hierarchies

	if hierarchies == nil {
		bookItem = BookItem{
			Level:   level,
			Name:    "-",
			TagID:   tagItem.TagID,
			TagName: tagMap[tagItem.TagID],
		}
		return bookItem
	}

	hierarchy := hierarchies[0]
	children := hierarchy.Children

	bookItem.Name = hierarchy.HierarchyName
	bookItem.Level = level
	bookItem.TagName = tagItem.TagName
	bookItem.TagID = tagItem.TagID
	bookItem.IsBook = false

	if len(children) > 0 {
		for _, child := range children {
			childBook := ParseHierarchies2(level+1, child, tagMap)
			bookItem.Children = append(bookItem.Children, childBook)
		}
	}
	return bookItem
}

func UpdateHierarchies2(bookBase *BookItem, tagMap map[string]string, docPDFList []DocPDFData) {
	// 添加docPDFMap中额外的数据
	for _, doc := range docPDFList {
		tagPath := strings.Split(doc.TagPath, "/")
		previousItem := bookBase
		currentItem := bookBase
		if tagPath[0] != bookBase.TagID {
			continue
		}

		for i := 1; i < len(tagPath); i++ {
			currentTagID := tagPath[i]
			previousItem = currentItem
			currentItem = nil
			flag := false
			if previousItem != nil && previousItem.Children != nil {
				for j, item := range previousItem.Children {
					if item.TagID == currentTagID {
						currentItem = &previousItem.Children[j]
						flag = true
						break
					}
				}
			}

			if flag {
				continue
			}

			newBookItem := BookItem{
				Level:    previousItem.Level + 1,
				Name:     tagMap[currentTagID],
				TagName:  tagMap[currentTagID],
				TagID:    currentTagID,
				Children: []BookItem{},
			}
			if i == len(tagPath)-1 {
				newBookItem.BookName = doc.Title
				newBookItem.BookID = doc.ID
				newBookItem.IsBook = true
			}

			previousItem.Children = append(previousItem.Children, newBookItem)
			for j, item := range previousItem.Children {
				if item.TagID == currentTagID {
					currentItem = &previousItem.Children[j]
					flag = true
					break
				}
			}
		}
	}
}

func FetchRawData2(name string, local bool) BookItem {
	tagData, dataList := ReadRawData(name, local)

	tagBase := ParseHierarchies(tagData)
	tagMap, _, docPDFList := ParseDataList(dataList)

	if len(tagBase.Hierarchies) > 0 && len(tagBase.Hierarchies[0].Children) > 0 {
		bookItem := ParseHierarchies2(1, tagBase.Hierarchies[0].Children[0], tagMap)
		bookItemBase := BookItem{
			Level:    0,
			Name:     tagBase.Hierarchies[0].HierarchyName,
			TagName:  "",
			TagID:    tagBase.TagID,
			Children: []BookItem{bookItem},
		}
		UpdateHierarchies2(&bookItemBase, tagMap, docPDFList)
		return bookItemBase
	}

	return BookItem{}
}

func Query2(bookItem BookItem) (string, []BookOption, []BookItem) {
	title := bookItem.Name
	children := bookItem.Children
	bookOptions := []BookOption{}

	if children != nil && children[0].IsBook {
		for _, child := range children {
			name := child.BookName
			name = strings.ReplaceAll(name, "•", "·")
			bookOptions = append(bookOptions, BookOption{child.BookID, "《" + name + "》"})
		}
		children = nil

	} else {
		for _, child := range children {
			name := child.TagName
			name = strings.ReplaceAll(name, "•", "·")
			name = strings.ReplaceAll(name, " ", "")
			bookOptions = append(bookOptions, BookOption{child.TagID, name})
		}
	}

	if bookItem.IsBook || bookItem.Level > 4 {
		optionNames := []string{}
		for _, option := range bookOptions {
			optionNames = append(optionNames, option.OptionName)
		}
		slog.Debug(fmt.Sprintf("Query result: level=%d, title=%s tag=%s/%s, book=%s/%s; optionNames=%d/%v", bookItem.Level, title, bookItem.TagName, bookItem.TagID, bookItem.BookName, bookItem.BookID, len(optionNames), optionNames))
	}
	return title, bookOptions, children
}
