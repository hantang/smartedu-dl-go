package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func saveJSONToFile(jsonData []byte, filePath string) error {
	slog.Debug("Save json data to " + filePath)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	// indentation (2 spaces)
	indentedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, indentedJSON, 0644)
}

func concatTagPath(tags []DocTag, dimIDOrders []string) string {
	dimToTag := make(map[string]string)
	for _, tag := range tags {
		dimToTag[tag.TagDim] = tag.TagID
	}

	// 按照指定顺序构建路径
	var pathParts []string
	for _, dimID := range dimIDOrders {
		if tagID, exists := dimToTag[dimID]; exists {
			pathParts = append(pathParts, tagID)
		}
	}
	return strings.Join(pathParts, "/")
}

func ParseData(data []byte) (map[string]string, map[string]DocPDFData, []DocPDFData) {
	var DocItemList []DocResourceItem
	if err := json.Unmarshal(data, &DocItemList); err != nil {
		return nil, nil, nil
	}

	tagMap := map[string]string{}
	docPDFMap := map[string]DocPDFData{}
	docPDFList := []DocPDFData{}

	// 拼接字段顺序 "tagView" 同步课资源视图 / 学段 / 年级 / 学科 / 版本  / 册次 / 新旧教材
	dimIDOrders := []string{"zxxxd", "zxxnj", "zxxxk", "zxxbb", "zxxcc", "zxxxjjc"}

	for _, item := range DocItemList {
		for _, tag := range item.TagList {
			tagMap[tag.TagID] = tag.TagName
		}

		tagPaths := item.TagPaths
		if tagPaths == nil {
			// 仅教材列表有tag_paths，课程没有
			tagPaths = []string{concatTagPath(item.TagList, dimIDOrders)}
		}

		for _, tagPath := range tagPaths {
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
		slog.Debug(fmt.Sprintf("partDocPDFList = %d", len(partDocPDFList)))

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
		slog.Warn(fmt.Sprintf("Error unmarshaling: %s", err))
	}
	return tagBase
}

func fetchJSONFile(url string, filePath string, local bool) ([]byte, error, bool) {
	slog.Debug(fmt.Sprintf("process path = %s / file = %s", url, filePath))
	if !local {
		slog.Debug("Fetch data from " + url)
		return FetchJsonData(url)
	}

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		return data, err, true
	} else {
		slog.Debug("Local file do not exists, try fetch data from " + url)
		data, err, status := FetchJsonData(url)
		// save json data
		saveJSONToFile(data, filePath)
		return data, err, status
	}
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

func readRawData(name string, local bool) ([]byte, [][]byte) {
	dataDir := TchMaterialInfo.Directory
	tagURL := TchMaterialInfo.Tag
	versionURL := TchMaterialInfo.Version
	if name == TAB_NAMES[2] {
		dataDir = SyncClassroomInfo.Directory
		tagURL = SyncClassroomInfo.Tag
		versionURL = SyncClassroomInfo.Version
	}

	var tagData []byte
	dataList := [][]byte{}

	tagPath := path.Join(dataDir, path.Base(tagURL))
	versionPath := path.Join(dataDir, path.Base(versionURL))

	tagData, err, statusOK := fetchJSONFile(tagURL, tagPath, local)
	if err != nil && statusOK {
		return tagData, dataList
	}

	versionData, err, statusOK := fetchJSONFile(versionURL, versionPath, local)
	if err != nil && statusOK {
		return tagData, dataList
	}

	urls, err := ParseURLsFromJSON(versionData)
	if err != nil {
		return tagData, dataList
	}

	for _, url := range urls {
		dataPath := path.Join(dataDir, path.Base(url))
		data, err, statusOK := fetchJSONFile(url, dataPath, local)
		if err != nil && statusOK {
			continue
		}
		dataList = append(dataList, data)
	}
	return tagData, dataList
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

		start := 1
		if tagPath[0] != bookBase.TagID {
			start = 0
		}

		for i := start; i < len(tagPath); i++ {
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

			if flag && i+1 < len(tagPath) {
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

			if flag {
				// 匹配，直接添加作为当前的子节点
				currentItem.Children = append(currentItem.Children, newBookItem)
				break
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
	tagData, dataList := readRawData(name, local)

	tagBase := ParseHierarchies(tagData)
	tagMap, _, docPDFList := ParseDataList(dataList)
	slog.Info(fmt.Sprintf("total docPDFList = %d", len(docPDFList)))

	if len(tagBase.Hierarchies) > 0 {
		count := len(tagBase.Hierarchies[0].Children)
		bookItems := []BookItem{}
		for index := range count {
			bookItem := ParseHierarchies2(1, tagBase.Hierarchies[0].Children[index], tagMap)
			bookItems = append(bookItems, bookItem)
		}

		bookItemBase := BookItem{
			Level:    0,
			Name:     tagBase.Hierarchies[0].HierarchyName,
			TagName:  "",
			TagID:    tagBase.TagID,
			Children: bookItems,
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

	if len(children) > 0 && children[0].IsBook {
		bookOptions = queryBooks(bookItem, []string{})
		children = nil
	} else {
		for _, child := range children {
			name := child.TagName
			name = strings.ReplaceAll(name, "•", "·")
			name = strings.ReplaceAll(name, " ", "")
			bookOptions = append(bookOptions, BookOption{child.TagID, name})
		}
	}

	return title, bookOptions, children
}

func queryBooks(bookItem BookItem, prefixList []string) []BookOption {
	bookOptions := []BookOption{}
	prefixList = append(prefixList, bookItem.TagName)
	for _, child := range bookItem.Children {
		if child.IsBook {
			name := child.BookName
			name = strings.ReplaceAll(name, "•", "·")
			fullname := "《" + name + "》"
			prefix := ""
			if len(prefixList) > 1 {
				prefix = "[" + strings.Join(prefixList, "-") + "] "
			}
			fullname = prefix + fullname
			bookOptions = append(bookOptions, BookOption{child.BookID, fullname})
		} else {
			more := queryBooks(child, prefixList)
			bookOptions = append(bookOptions, more...)
		}
	}
	return bookOptions
}

func ParseCourseID(courseID string) []CourseToc {
	// b7062df1-f929-458e-964c-d778f89ca255\
	server := SERVER_LIST[rand.Intn(len(SERVER_LIST))]
	var courseInfo []DataCourseInfo        //
	var courseChapters []DataCourseChapter // 课程单元 array + tree
	var courseToc []CourseToc

	pattern := "https://%s.ykt.cbern.com.cn/zxx/ndrs/national_lesson/teachingmaterials/%s/resources/parts.json"
	pattern2 := "https://%s.ykt.cbern.com.cn/zxx/ndrv2/national_lesson/trees/%s.json"

	url := fmt.Sprintf(pattern, server, courseID)
	slog.Debug(fmt.Sprintf("URL = %s", url))
	data, err, _ := FetchJsonData(url) // parts.json
	if err != nil {
		return courseToc
	}

	var urls []string
	err = json.Unmarshal(data, &urls) // part_100.json
	if err != nil {
		slog.Warn(fmt.Sprintf("error = %s", err))
		return courseToc
	}
	slog.Debug(fmt.Sprintf("urls = %s", urls))

	for _, url := range urls {
		data, err, _ := FetchJsonData(url)
		if err != nil {
			continue
		}
		var units []DataCourseInfo
		if err := json.Unmarshal(data, &units); err != nil {
			continue
		}
		courseInfo = append(courseInfo, units...)
	}
	slog.Debug(fmt.Sprintf("courseInfo = %d; ids = %d", len(courseInfo), len(courseInfo[0].TeachIDs)))

	if len(courseInfo) == 0 || len(courseInfo[0].TeachIDs) == 0 {
		return courseToc
	}

	teachID := courseInfo[0].TeachIDs[0] // = tree_id
	url = fmt.Sprintf(pattern2, server, teachID)
	data, err, _ = FetchJsonData(url)
	if err != nil {
		return courseToc
	}
	if err := json.Unmarshal(data, &courseChapters); err != nil {
		return courseToc
	}

	courseToc = initChapters(courseInfo, courseChapters)
	return courseToc
}

func createCourseDict(courseInfo []DataCourseInfo) map[string]DataCourseInfo {
	courseDict := make(map[string]DataCourseInfo)
	for _, course := range courseInfo {
		chapters := course.ChapterPaths
		if course.ResourceType == "national_lesson" || course.ResourceType == "elite_lesson" {
			for _, chapter := range chapters {
				courseDict[chapter] = course
			}
		}
	}
	return courseDict
}

func getChapterNode(courseChapters []DataCourseChapter, courseDict map[string]DataCourseInfo, parentTitles []string) []CourseItem {
	var courseItems []CourseItem
	for _, chapter := range courseChapters {
		newParentTitles := append(parentTitles, chapter.Title)
		if chapter.Children == nil {
			if value, ok := courseDict[chapter.NodePath]; ok {
				var parent []string
				if len(parentTitles) > 0 {
					parent = append(parent, parentTitles[len(parentTitles)-1])
				}
				fullTitle := strings.Join(append(parent, chapter.Title), " / ") // value.Title

				item := CourseItem{
					Title:        fullTitle,
					NodeTitle:    chapter.Title,
					NodeParents:  parentTitles,
					NodeID:       chapter.ID,
					NodePath:     chapter.NodePath,
					ResourceType: value.ResourceType,
					CourseID:     value.ID,
					CourseTitle:  value.Title,
				}
				courseItems = append(courseItems, item)
			}
		} else {
			more := getChapterNode(chapter.Children, courseDict, newParentTitles)
			courseItems = append(courseItems, more...)
		}
	}
	return courseItems
}

func initChapters(courseInfo []DataCourseInfo, courseChapters []DataCourseChapter) []CourseToc {
	var courseToc []CourseToc
	courseDict := createCourseDict(courseInfo)
	for index, chapter := range courseChapters {
		items := getChapterNode(chapter.Children, courseDict, []string{})
		toc := CourseToc{
			Index:    index,
			Title:    chapter.Title,
			Children: items,
		}
		courseToc = append(courseToc, toc)
	}
	return courseToc
}
