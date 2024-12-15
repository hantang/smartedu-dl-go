package dl

import (
	"fmt"
	"log/slog"
	"strings"
)

func ParseHierarchies2(level int, tagItem TagItem, tagMap map[string]string, docPDFMap map[string]DocPDFData) BookItem {
	var bookItem BookItem
	hierarchies := tagItem.Hierarchies

	if hierarchies == nil {
		bookItem = BookItem{
			Level:   level,
			Name:    "-",
			TagID:   tagItem.TagID,
			TagName: tagMap[tagItem.TagID],
		}
		if val, ok := docPDFMap[tagItem.TagID]; ok {
			bookItem.IsBook = true
			bookItem.BookID = val.ID
			bookItem.BookName = val.Title
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
			childBook := ParseHierarchies2(level+1, child, tagMap, docPDFMap)
			bookItem.Children = append(bookItem.Children, childBook)
		}
	} else {
		for _, hiddenTagID := range hierarchy.Ext.HiddenTags {
			if val, ok := docPDFMap[hiddenTagID]; ok {
				childBook := BookItem{
					Level:    level + 1,
					Name:     "-",
					TagID:    hiddenTagID,
					TagName:  tagMap[hiddenTagID],
					IsBook:   true,
					BookID:   val.ID,
					BookName: val.Title,
				}
				bookItem.Children = append(bookItem.Children, childBook)
			}
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
	tagMap, docPDFMap, docPDFList := ParseDataList(dataList)

	if len(tagBase.Hierarchies) > 0 && len(tagBase.Hierarchies[0].Children) > 0 {
		bookItem := ParseHierarchies2(1, tagBase.Hierarchies[0].Children[0], tagMap, docPDFMap)
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
		// sort.Slice(bookOptions, func(i, j int) bool {
		// 	return bookOptions[i].OptionName < bookOptions[j].OptionName
		// })
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
