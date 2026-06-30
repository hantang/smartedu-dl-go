package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	serverKey      = "server"
	contentTypeKey = "contentType"
	// resourceTypeKey = "resourceType"
)

func ValidURL(link string) bool {
	if link == "" || !strings.HasPrefix(link, "http") {
		return false
	}

	parsedURL, err := url.Parse(link)
	if err != nil {
		return false
	}

	path := parsedURL.Path
	// 允许直接输入下载资源链接
	if strings.Contains(path, RESOURCES_PATH) {
		return true
	}
	// TODO 放宽限制
	// if _, ok := RESOURCES_MAP[path]; !ok {
	// 	return false
	// }
	return true
}

func buildURLParamDict(queryParams url.Values, configParams []string, random bool) map[string]string {

	keys := append([]string{}, configParams...)
	keys = append(keys, contentTypeKey)

	m := make(map[string]string, len(keys)+1)

	for _, k := range keys {
		m[k] = queryParams.Get(k)
	}

	if len(SERVER_LIST) > 0 {
		idx := 0
		if random {
			idx = rand.Intn(len(SERVER_LIST))
		}
		m[serverKey] = SERVER_LIST[idx]
	}

	return m
}

func parseResourceURL(path string, queryParams url.Values, audio bool, random bool, useBackup bool) ([]string, error) {
	var configURLList []string
	configInfo, ok := RESOURCES_MAP[path]
	if !ok {
		return configURLList, fmt.Errorf("invalid url path: %s", path)
	}

	// 忽略
	contentTypeValue := queryParams.Get(contentTypeKey)
	if path == "/tchMaterial/detail" && contentTypeValue != "assets_document" {
		message := fmt.Sprintf("invalid params %s: %s", contentTypeKey, contentTypeValue)
		slog.Warn(message)
		return configURLList, fmt.Errorf("error %v", message)
	}

	// 参数列表
	paramDict := buildURLParamDict(queryParams, configInfo.params, random)
	slog.Debug("paramDict = " + fmt.Sprintf("%v", paramDict))

	var paramValues []any
	paramValues = append(paramValues, paramDict[serverKey])
	for _, k := range configInfo.params {
		paramValues = append(paramValues, paramDict[k])
	}
	configURL := fmt.Sprintf(configInfo.resources.basic, paramValues...)
	configURLList = append(configURLList, configURL)

	if useBackup {
		for _, backupURL := range configInfo.resources.backup {
			configURLList = append(configURLList, fmt.Sprintf(backupURL, paramValues...))
		}
	}

	if audio && configInfo.resources.audio != "" {
		configURLList = append(configURLList, fmt.Sprintf(configInfo.resources.audio, paramValues...))
	}
	return configURLList, nil
}

func parseExtraResourceURL(queryParams url.Values, random bool) ([]string, error) {
	const path = "/"
	var configURLList []string
	configInfo, ok := RESOURCES_MAP[path]
	if !ok {
		return configURLList, fmt.Errorf("invalid url path: %s", path)
	}
	contentTypeValue := queryParams.Get(contentTypeKey)

	// 参数列表
	paramDict := buildURLParamDict(queryParams, configInfo.params, random)
	slog.Debug("paramDict = " + fmt.Sprintf("%v", paramDict))

	var paramValues []any
	paramValues = append(paramValues, paramDict[serverKey])
	for _, k := range configInfo.params {
		paramValues = append(paramValues, paramDict[k])
	}

	urlTemplate := configInfo.resources.basic
	if contentTypeValue == "thematic_course" && len(configInfo.resources.backup) > 0 {
		urlTemplate = configInfo.resources.backup[0]
	}
	configURL := fmt.Sprintf(urlTemplate, paramValues...)
	configURLList = append(configURLList, configURL)

	return configURLList, nil
}

func parseURL(link string, audio bool, random bool, useBackup bool) ([]string, error) {
	// 解析 smartedu.cn 详情页链接，得到json资源链接
	var configURLList []string
	parsedURL, err := url.Parse(link)
	if err != nil {
		return configURLList, err
	}

	// 提取主机、路径和查询参数
	path := parsedURL.Path
	queryParams := parsedURL.Query()
	if strings.Contains(path, RESOURCES_PATH) { // 资源链接，不再额外解析
		configURLList = append(configURLList, link)
		return configURLList, nil
	}

	_, ok := RESOURCES_MAP[path]
	if ok {
		return parseResourceURL(path, queryParams, audio, random, useBackup)
	} else if useBackup {
		return parseExtraResourceURL(queryParams, random)
	}

	return configURLList, nil
}

func parseURLList(links []string, audio bool, random bool, useBackup bool) []string {
	var configURLList []string
	for _, link := range links {
		output, err := parseURL(link, audio, random, useBackup)
		if err != nil {
			slog.Debug(fmt.Sprintf("parse link error: %v", err))
			continue
		}
		configURLList = append(configURLList, output...)
	}
	return configURLList
}

func convertURL(rawLink string, isClean bool) string {
	// 备用解析，可能是旧版教材，不一定有效
	// 原始：https://r3-ndr-private.ykt.cbern.com.cn/edu_product/esp/assets/<id>.pkg/<title>_<毫秒时间戳>.pdf
	// => https://r3-ndr.ykt.cbern.com.cn/edu_product/esp/assets/<id>.pkg/pdf.pdf
	// => https://r3-ndr.ykt.cbern.com.cn/edu_product/esp/assets/<id>.pdf

	link := rawLink
	slog.Debug(fmt.Sprintf("Raw URL = %s", link))

	// link = regexp.MustCompile(`[^/]+\.pdf$`).ReplaceAllString(link, "pdf.pdf")
	link = regexp.MustCompile(`(/[\w\-]+)\.pkg/[\w\-]+\.pdf$`).ReplaceAllString(link, "${1}.pdf")
	slog.Debug(fmt.Sprintf("Update URL = %s", link))

	if isClean {
		link = strings.ReplaceAll(link, "ndr-private.", "ndr.")
		slog.Debug(fmt.Sprintf("Cleaned URL = %s", link))
	}
	return link
}

// 提取教师名称并拼接
func getTeacherNames(r ResourceItemExt) string {
	// slog.Debug(fmt.Sprintf("Teacher %v", r.TeacherList))
	var names []string
	for _, teacher := range r.TeacherList {
		if teacher.Name != "" {
			names = append(names, teacher.Name)
		}
	}
	return strings.Join(names, " ")
}

func concatFullTitle(title string, bookName string, schoolName string, teacherNames string) string {
	// 完整格式：教材名-课程名 (学校_教师)
	baseTitle := title
	// 教材名
	if bookName != "" {
		baseTitle = bookName + "-" + title
	}

	// 处理学校名称和教师列表
	var suffixParts []string
	if schoolName != "" {
		suffixParts = append(suffixParts, schoolName)
	}
	if teacherNames != "" {
		suffixParts = append(suffixParts, teacherNames)
	}

	if len(suffixParts) > 0 {
		baseTitle = fmt.Sprintf("%s (%s)", baseTitle, strings.Join(suffixParts, "_"))
	}

	return baseTitle
}

func getFirstItemResource[T any](relations any) (items []T) {
	// itemExt.Relations 中获得候选
	v := reflect.ValueOf(relations)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() != reflect.Slice {
			continue
		}

		slog.Debug("Relations",
			"name", fieldType.Name,
			"json", fieldType.Tag.Get("json"),
			"len", field.Len(),
		)

		if field.Len() == 0 {
			continue
		}

		items, ok := field.Interface().([]T)
		if !ok {
			continue
		}
		return items
	}

	return nil
}

func parseResourceItems(data []byte, tiFormatList []string, random bool) ([]LinkData, error) {
	// 解析资源文件（json格式），得到最终下来文件的链接
	var result []LinkData
	var items []ResourceItem

	// 尝试解析为ResourceItemExt (课程包)
	var itemExt ResourceItemExt
	teacherNames := ""
	schoolName := ""
	bookName := ""

	if err := json.Unmarshal(data, &itemExt); err == nil {
		slog.Debug(fmt.Sprintf("CustomProperties %v", itemExt.CustomProperties))
		teacherNames = getTeacherNames(itemExt)
		schoolName = itemExt.CustomProperties.SchoolName
		bookName = itemExt.CustomProperties.BookInfo.Name

		slog.Debug("Parse into ResourceItemExt")
		tempItems := getFirstItemResource[ResourceItem](itemExt.Relations)
		if tempItems != nil {
			items = tempItems
		}
	}

	// 如果不是ResourceItemExt，尝试解析为ResourceItem数组（教材类）
	if len(items) == 0 {
		slog.Debug("Parse into ResourceItem list")
		if err := json.Unmarshal(data, &items); err != nil {
			// 如果不是数组，尝试解析为单个ResourceItem
			slog.Debug("Parse into single ResourceItem")
			var item ResourceItem
			if err := json.Unmarshal(data, &item); err != nil {
				return nil, err
			}
			items = []ResourceItem{item}
		}
	}

	slog.Debug(fmt.Sprintf("Extract items = %d", len(items)))
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource items")
	}

	// 处理每个ResourceItem
	for i, item := range items {
		slog.Debug(fmt.Sprintf("item = %v", item))
		// 补充额外信息，尽量避免标题重复
		title := item.CustomProperties.OriginalTitle
		if title == "" {
			title = item.Title
		}
		if item.CustomProperties.AliasName != "" {
			if title != "" {
				title = title + "-" + item.CustomProperties.AliasName
			} else {
				title = item.CustomProperties.AliasName
			}
		}
		if title == "" {
			if item.ResourceType != "" {
				title = fmt.Sprintf("%s-%03d", item.ResourceType, i)
			} else {
				title = fmt.Sprintf("%s-%03d", "未命名", i)
			}
		}
		slog.Debug(fmt.Sprintf("title = %s", title))
		var rawLink string
		var format string
		var size int64

		for _, tiItem := range item.TiItems {
			format = tiItem.TiFormat
			if tiItem.TiFormat == "folder" {
				if v, ok := MIME_TO_FORMAT[tiItem.LcTiFormat]; ok {
					format = v
				}
			}
			slog.Debug(fmt.Sprintf("formats = %s, %s; file suffix = %s", tiItem.TiFormat, tiItem.LcTiFormat, format))
			if !slices.Contains(tiFormatList, format) || len(tiItem.TiStorages) == 0 {
				continue
			}

			// 随机选择其中一个链接
			randomIndex := 0
			if random {
				randomIndex = rand.Intn(len(tiItem.TiStorages))
			}
			rawLink = tiItem.TiStorages[randomIndex]
			size = tiItem.TiSize
			if len(tiItem.CustomProperties.Requirements) > 0 {
				for _, reqItem := range tiItem.CustomProperties.Requirements {
					slog.Debug(fmt.Sprintf("reqItem = %v", reqItem))
					// 视频大小
					if reqItem.Name == "total_size" && len(reqItem.Value) > 0 {
						value, err := strconv.ParseInt(reqItem.Value[0], 10, 64) // 返回 int 类型
						if err == nil {
							size = value
						}
					}
					if tiItem.TiFormat == "folder" {
						if reqItem.Name == "fileRange" && reqItem.Type == "RANGE" && len(reqItem.Value) > 0 {
							rawLink = strings.TrimRight(rawLink, "/") + "/" + strings.TrimLeft(reqItem.Value[0], "/")
						}
						if strings.HasSuffix(rawLink, "/image") {
							rawLink = ""
						}
					}
				}
			}

			if title == "" {
				title = fmt.Sprintf("%s-%03d", strings.ToUpper(format), i)
			}
			if rawLink != "" {
				break
			}
		}

		fullTitle := concatFullTitle(title, bookName, schoolName, teacherNames)
		if rawLink != "" {
			linkData := LinkData{
				Format:    format,
				Title:     fullTitle,
				ID:        item.ID,
				RawURL:    rawLink,
				BackupURL: convertURL(rawLink, true), // 备用下载链接
				Size:      size,
			}
			result = append(result, linkData)
			slog.Debug(fmt.Sprintf("format = %s, linkData = %v", format, linkData))
		}
	}

	slog.Debug(fmt.Sprintf("Extract result items = %d", len(result)))
	return result, nil
}

func parsePaperResourceItems(data []byte) ([]LinkData, error) {
	// 练习（试卷） 或者 container_id 字段，请求data.json获得pdf
	var result []LinkData
	var resourceItem ResourceItem
	if err := json.Unmarshal(data, &resourceItem); err != nil {
		return nil, err
	}

	resourceType := resourceItem.ResourceType // "试卷"
	dataURL := fmt.Sprintf("%s/xedu_cs_paper_bank/api_static/papers/%s_%s/data.json", PAPER_SERVER, resourceItem.ContainerID, resourceItem.ID)
	slog.Debug(fmt.Sprintf("resourceType = %s, dataURL = %v", resourceType, dataURL))
	dataResult, err, statusOK := FetchJsonData(dataURL)
	if err != nil || !statusOK {
		slog.Warn(fmt.Sprintf("fetch data error: %v / status=%v", err, statusOK))
		return nil, err
	}

	var paperItem PaperItem
	if err := json.Unmarshal(dataResult, &paperItem); err != nil {
		return nil, err
	}

	pdfLinks := []string{paperItem.PDF_MAIN_LINK, paperItem.PDF_FULL_LINK}
	for _, urlPath := range pdfLinks {
		slog.Info(fmt.Sprintf("pdfLink = %s", urlPath))
		if strings.HasPrefix(urlPath, "/") {
			downloadURL := PAPER_SERVER + urlPath
			filename := path.Base(urlPath)
			name := strings.TrimSuffix(filename, path.Ext(filename))
			if name == "" {
				name = paperItem.Title
			}
			if resourceType != "" {
				name = resourceType + "-" + name
			}
			slog.Debug(fmt.Sprintf("name = %s, downloadURL = %s", name, downloadURL))
			result = append(result, LinkData{
				Format:    "pdf",
				Title:     name,
				ID:        paperItem.ID,
				RawURL:    downloadURL,
				BackupURL: downloadURL,
				Size:      -1,
			})
		}
	}
	slog.Debug(fmt.Sprintf("Extract result items = %d", len(result)))
	return result, nil
}

func getResourceItem(link string) (LinkData, error) {
	// 直接解析资源链接
	ext := filepath.Ext(link)
	format := strings.TrimPrefix(ext, ".")
	title := strings.ToTitle(format)
	id := ""

	pattern := `/assets/([\w\-]+)`
	matches := regexp.MustCompile(pattern).FindStringSubmatch(link)
	if len(matches) > 0 {
		id = matches[1]
	}

	pattern = `/zh-CN/(\d+)/(?:transcode/)?(\w+/)?[\w\-]+\.(\w+)$`
	matches = regexp.MustCompile(pattern).FindStringSubmatch(link)

	if len(matches) > 0 {
		title_id := matches[1]
		genre := strings.Trim(matches[2], "/")
		if genre == "" {
			genre = matches[3]
		}
		title = fmt.Sprintf("%s-%s", strings.ToTitle(genre), title_id)
	}

	backupLink := convertURL(link, true)
	result := LinkData{
		Format:    format,
		Title:     title,
		ID:        id,
		RawURL:    link,
		BackupURL: backupLink,
		Size:      -1,
	}
	return result, nil
}

func removeDuplicates(result []LinkData) []LinkData {
	// 判断URL路径（不包括域名和参数等）过滤重复
	counts := make(map[string]int)
	var unique []LinkData

	for _, item := range result {
		parsedURL, err := url.Parse(item.BackupURL) // item.ID
		if err != nil {
			continue
		}

		key := parsedURL.Path
		if counts[key] == 0 {
			unique = append(unique, item)
		}
		counts[key]++
	}

	return unique
}

func renameDuplicates(result []LinkData) []LinkData {
	counts := make(map[string]int)
	var unique []LinkData

	for _, item := range result {
		key := item.Title
		if counts[key] > 0 {
			newTitle := ""
			if item.ID != "" {
				newTitle = fmt.Sprintf("%s_%s", item.Title, item.ID)
			} else {
				index := 1
				for {
					newTitle = fmt.Sprintf("%s (%d)", item.Title, index)
					if counts[newTitle] == 0 {
						break
					}
					index += 1
				}
			}

			item.Title = newTitle
			key = newTitle
		}
		unique = append(unique, item)
		counts[key]++
	}

	return unique
}

func ExtractResources(links []string, formatList []string, random bool, useBackup bool, isParse bool) []LinkData {
	var result []LinkData

	var audio = false
	for _, format := range formatList {
		if format == "mp3" || format == "ogg" {
			audio = true
			break
		}
	}
	slog.Debug(fmt.Sprintf("formats=%v audio=%v random=%v backup=%v", formatList, audio, random, useBackup))
	configURLList := links
	if isParse {
		configURLList = parseURLList(links, audio, random, useBackup)
	}
	slog.Debug(fmt.Sprintf("configURLList is %v", len(configURLList)))

	for _, url := range configURLList {
		slog.Debug("config url = " + url)

		// 是否直接资源链接
		if strings.Contains(url, RESOURCES_PATH) {
			resource, err := getResourceItem(url)
			if err == nil {
				if slices.Contains(formatList, resource.Format) {
					result = append(result, resource)
				}
				continue
			}
		}

		data, err, statusOK := FetchJsonData(url)
		if err != nil || !statusOK {
			slog.Warn(fmt.Sprintf("fetch data error: %v / status=%v", err, statusOK))
			continue
		}

		// TODO
		if strings.Contains(url, "/examinationpapers") {
			slog.Debug(fmt.Sprintf("formatList = %v", formatList))
			if slices.Contains(formatList, "pdf") {
				resources, err := parsePaperResourceItems(data)
				if err != nil {
					slog.Warn(fmt.Sprintf("parse paper resource error: %v", err))
					continue
				}
				result = append(result, resources...)
			}
		} else {
			resources, err := parseResourceItems(data, formatList, random)
			if err != nil {
				slog.Warn(fmt.Sprintf("parse resource data error: %v", err))
				continue
			}
			result = append(result, resources...)
		}
	}

	// 去重
	unique := removeDuplicates(result)
	if len(result) != len(unique) {
		slog.Info(fmt.Sprintf("After deduplication resources = %d -> %d", len(result), len(unique)))
	}
	// 重名处理
	renamed := renameDuplicates(unique)
	return renamed
}

func GenerateURLFromID(linkItems []LinkItem) []string {
	// book_id/link转化成URL
	metaInfoList := []ResourceMetaInfo{InputInfo, TchMaterialInfo, SyncClassroomInfo, EliteSyncClassroomInfo}
	urls := []string{}

	for _, linkItem := range linkItems {
		for _, metaInfo := range metaInfoList {
			if linkItem.Type == metaInfo.Type {
				example_url := metaInfo.Detail
				url := linkItem.Link
				if example_url != "" {
					url = fmt.Sprintf(example_url, url)
				}
				urls = append(urls, url)
				break
			}
		}
	}

	return urls
}

func GenerateURLFromID2(linkItems []LinkItem) []string {
	urls := []string{}
	example_url := ReadingLibraryInfo.Detail
	for _, linkItem := range linkItems {
		url := fmt.Sprintf(example_url, linkItem.Link)
		urls = append(urls, url)
	}
	return urls
}
