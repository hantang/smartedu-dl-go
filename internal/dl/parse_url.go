package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func cleanURL(link string) string {
	return strings.ReplaceAll(link, "ndr-private.", "ndr.")
}

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
	if _, ok := RESOURCES_MAP[path]; !ok {
		return false
	}
	return true
}

func parseURL(link string, audio bool, random bool, useBackup bool) ([]string, error) {
	// 解析 smartedu.cn 详情页链接，得到json资源链接
	var configURLList []string
	parsedURL, err := url.Parse(link)
	if err != nil {
		return configURLList, err
	}

	// 提取主机、路径和查询参数
	// host := parsedURL.Host
	path := parsedURL.Path
	queryParams := parsedURL.Query()
	if strings.Contains(path, RESOURCES_PATH) { // 资源链接，不再额外解析
		configURLList = append(configURLList, link)
		return configURLList, nil
	}

	configInfo, ok := RESOURCES_MAP[path]
	if !ok {
		return configURLList, fmt.Errorf("invalid url path: %s", path)
	}

	contentTypeKey := "contentType"
	serverKey := "server"
	var paramList []string = append(configInfo.params, contentTypeKey)
	paramDict := map[string]string{}
	for _, key := range paramList {
		paramDict[key] = queryParams.Get(key)
	}
	if path == "/tchMaterial/detail" && paramDict[contentTypeKey] != "assets_document" {
		slog.Warn(fmt.Sprintf("Error %s = %s. Ignore", contentTypeKey, paramDict[contentTypeKey]))
		return configURLList, fmt.Errorf("invalid %s: %s", contentTypeKey, paramDict[contentTypeKey])
	}

	if random {
		paramDict[serverKey] = SERVER_LIST[rand.Intn(len(SERVER_LIST))]
	} else {
		paramDict[serverKey] = SERVER_LIST[0]
	}
	slog.Debug("paramDict = " + fmt.Sprintf("%v", paramDict))

	configURL := fmt.Sprintf(configInfo.resources.basic, paramDict[serverKey], paramDict[configInfo.params[0]])
	configURLList = append(configURLList, configURL)
	if useBackup {
		for _, backupURL := range configInfo.resources.backup {
			moreConfigURL := fmt.Sprintf(backupURL, paramDict[serverKey], paramDict[configInfo.params[0]])
			configURLList = append(configURLList, moreConfigURL)
		}
	}

	if audio && configInfo.resources.audio != "" {
		audioURL := fmt.Sprintf(configInfo.resources.audio, paramDict[serverKey], paramDict[configInfo.params[0]])
		slog.Debug(fmt.Sprintf("audioURL = %s", audioURL))
		configURLList = append(configURLList, audioURL)
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

func convertURL(rawLink string, randomIndex int) string {
	// link ~~去除“-doc-private”~~
	// => https://r3-ndr.ykt.cbern.com.cn/edu_product/esp/assets/<教材代码>.pkg/pdf.pdf
	slog.Debug(fmt.Sprintf("Raw URL = %s", rawLink))
	link := rawLink
	link = regexp.MustCompile(`[^/]+\.pdf$`).ReplaceAllString(link, "pdf.pdf") // 可能是旧版教材
	link = regexp.MustCompile(`ndr-(doc-)?private`).ReplaceAllString(link, "ndr")

	slog.Debug(fmt.Sprintf("Update URL = %s", link))
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

	return baseTitle + ""
}

func parseResourceItems(data []byte, tiFormatList []string, random bool) ([]LinkData, error) {
	// 解析资源文件（json格式），得到最终下来文件的链接
	var result []LinkData
	var items []ResourceItem

	// 尝试解析为ResourceItemExt (课程类)
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
		tempItems := [][]ResourceItem{
			itemExt.Relations.NationalCourseResource,
			itemExt.Relations.EliteCourseResource,
		}
		for _, temp := range tempItems {
			if len(temp) > 0 {
				items = temp
				break
			}
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

	// 处理每个ResourceItem
	for i, item := range items {
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

		var link string
		var rawLink string
		var format string
		var size int64

		for _, tiItem := range item.TiItems {
			if !slices.Contains(tiFormatList, tiItem.TiFormat) || len(tiItem.TiStorages) == 0 {
				continue
			}

			// 随机选择其中一个链接
			randomIndex := 0
			if random {
				randomIndex = rand.Intn(len(tiItem.TiStorages))
			}
			rawLink = tiItem.TiStorages[randomIndex]
			link = convertURL(rawLink, randomIndex)
			if link == "" {
				continue
			}

			format = tiItem.TiFormat
			size = tiItem.TiSize // TODO
			if len(tiItem.CustomProperties.Requirements) > 0 {
				for _, reqItem := range tiItem.CustomProperties.Requirements {
					// 视频大小
					if reqItem.Name == "total_size" {
						value, err := strconv.ParseInt(reqItem.Value, 10, 64) // 返回 int 类型
						if err == nil {
							size = value
						}
					}
				}
			}

			if title == "" {
				title = fmt.Sprintf("%s-%03d", strings.ToUpper(format), i)
			}
			if link != "" {
				break
			}
		}
		fullTitle := concatFullTitle(title, bookName, schoolName, teacherNames)
		slog.Debug(fmt.Sprintf("title: %s %s %s %s %s", title, bookName, schoolName, teacherNames, fullTitle))
		if link != "" {
			link = cleanURL(link)
			result = append(result, LinkData{
				Format: format,
				Title:  fullTitle,
				ID:     item.ID,
				URL:    link,
				RawURL: rawLink,
				Size:   size,
			})
		}
	}

	return result, nil
}

func getResourceItem(link string) (LinkData, error) {
	// 直接解析资源链接
	ext := filepath.Ext(link)
	format := strings.TrimPrefix(ext, ".")
	title := strings.ToTitle(format)
	pattern := `/assets/([\w\-]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(link)
	id := ""
	if len(matches) > 0 {
		id = matches[1]
	}

	pattern = `/zh-CN/(\d+)/(?:transcode/)?(\w+/)?[\w\-]+\.(\w+)$`
	re = regexp.MustCompile(pattern)
	matches = re.FindStringSubmatch(link)

	if len(matches) > 0 {
		title_id := matches[1]
		genre := strings.Trim(matches[2], "/")
		if genre == "" {
			genre = matches[3]
		}
		title = fmt.Sprintf("%s-%s", strings.ToTitle(genre), title_id)
	}

	result := LinkData{
		Format: format,
		Title:  title,
		ID:     id,
		URL:    link,
		RawURL: link,
		Size:   -1,
	}
	return result, nil
}

func removeDuplicates(result []LinkData) []LinkData {
	// 判断URL路径（不包括域名和参数等）过滤重复
	counts := make(map[string]int)
	var unique []LinkData

	for _, item := range result {
		parsedURL, err := url.Parse(item.URL) // item.ID
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

func ExtractResources(links []string, formatList []string, random bool, useBackup bool) []LinkData {
	var result []LinkData

	var audio = false
	for _, format := range formatList {
		if format == "mp3" || format == "ogg" {
			audio = true
			break
		}
	}
	slog.Debug(fmt.Sprintf("formats=%v audio=%v random=%v backup=%v", formatList, audio, random, useBackup))

	configURLList := parseURLList(links, audio, random, useBackup)
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
		resources, err := parseResourceItems(data, formatList, random)
		if err != nil {
			slog.Warn(fmt.Sprintf("parse resource data error: %v", err))
			continue
		}
		result = append(result, resources...)
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
