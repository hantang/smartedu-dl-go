package dl

import (
	"crypto/sha256"
	"encoding/hex"
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

	if path == AIEducationListPath {
		return parseAIEducationListURL(parsedURL, random)
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

func pickServer(random bool) string {
	if random {
		return SERVER_LIST[rand.Intn(len(SERVER_LIST))]
	}
	return SERVER_LIST[0]
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func buildLibraryAdapterIndexURL(server string, libraryID string) string {
	adapterPath := fmt.Sprintf("/v1/libraries/%s/contents/actions/full/adapter", libraryID)
	hash := sha256Hex(adapterPath + "?sort_type=3")
	return fmt.Sprintf(
		"https://%s.ykt.cbern.com.cn/%s/api/zh-CN/%s/elearning_library%s/%s.json",
		server,
		AIEducationLibraryService,
		AIEducationLibraryAppID,
		adapterPath,
		hash,
	)
}

func fetchAIEducationListItems(indexURL string) ([]LibraryContentItem, error) {
	data, err, statusOK := FetchJsonData(indexURL)
	if err != nil || !statusOK {
		return nil, fmt.Errorf("fetch ai education list index error: %v / status=%v", err, statusOK)
	}

	var library DataLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		var items []LibraryContentItem
		if itemErr := json.Unmarshal(data, &items); itemErr != nil {
			return nil, err
		}
		return items, nil
	}

	parsedURL, err := url.Parse(indexURL)
	if err != nil {
		return nil, err
	}
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host
	var items []LibraryContentItem
	for _, file := range library.Files {
		fileURL := file
		if strings.HasPrefix(file, "/") {
			fileURL = baseURL + file
		} else if !strings.HasPrefix(file, "http") {
			fileURL = baseURL + "/" + file
		}

		partData, err, statusOK := FetchJsonData(fileURL)
		if err != nil || !statusOK {
			slog.Warn(fmt.Sprintf("fetch ai education list file error: %v / status=%v", err, statusOK))
			continue
		}
		var partItems []LibraryContentItem
		if err := json.Unmarshal(partData, &partItems); err != nil {
			slog.Warn(fmt.Sprintf("parse ai education list file error: %v", err))
			continue
		}
		items = append(items, partItems...)
	}
	return items, nil
}

func splitDefaultTags(raw string) []string {
	raw = unescapeQueryValue(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, "/")
	tags := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.EqualFold(part, "all") {
			continue
		}
		tags = append(tags, part)
	}
	return tags
}

func unescapeQueryValue(raw string) string {
	for range 2 {
		decoded, err := url.QueryUnescape(raw)
		if err != nil || decoded == raw {
			break
		}
		raw = decoded
	}
	return raw
}

func parseEmbeddedListQuery(raw string) url.Values {
	values := url.Values{}
	if raw == "" {
		return values
	}

	parsed, err := url.ParseQuery(raw)
	if err == nil {
		return parsed
	}

	for _, part := range strings.Split(raw, "&") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values.Add(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	return values
}

func parseAIEducationListParams(values url.Values) (string, []string, error) {
	contentID := unescapeQueryValue(strings.TrimSpace(values.Get("content_id")))
	if contentID == "" {
		return "", nil, fmt.Errorf("missing content_id")
	}

	defaultTagCandidates := []string{values.Get("defaultTag")}
	libraryID := contentID
	if cleanID, embeddedQuery, found := strings.Cut(contentID, "?"); found {
		libraryID = cleanID
		embeddedValues := parseEmbeddedListQuery(embeddedQuery)
		defaultTagCandidates = append(defaultTagCandidates, embeddedValues.Get("defaultTag"))
	} else if cleanID, embeddedQuery, found := strings.Cut(contentID, "&"); found {
		libraryID = cleanID
		embeddedValues := parseEmbeddedListQuery(embeddedQuery)
		defaultTagCandidates = append(defaultTagCandidates, embeddedValues.Get("defaultTag"))
	}

	libraryID = strings.Trim(strings.TrimSpace(libraryID), "/")
	if libraryID == "" {
		return "", nil, fmt.Errorf("missing content_id")
	}

	var selectedTags []string
	for _, candidate := range defaultTagCandidates {
		tags := splitDefaultTags(candidate)
		if len(tags) > len(selectedTags) {
			selectedTags = tags
		}
	}
	return libraryID, selectedTags, nil
}

func containsTag(item LibraryContentItem, tag string) bool {
	if tag == "" {
		return true
	}
	for _, itemTag := range item.Tags {
		if itemTag.ID == tag || itemTag.Code == tag || itemTag.Title == tag {
			return true
		}
	}
	return false
}

func filterAIEducationVideoItems(items []LibraryContentItem, selectedTags []string) ([]LibraryContentItem, string) {
	allVideos := []LibraryContentItem{}
	for _, item := range items {
		if isVideoLibraryContent(item) {
			allVideos = append(allVideos, item)
		}
	}
	if len(selectedTags) == 0 {
		return allVideos, ""
	}

	for i := len(selectedTags) - 1; i >= 0; i-- {
		tag := selectedTags[i]
		videoItems := []LibraryContentItem{}
		for _, item := range allVideos {
			if containsTag(item, tag) {
				videoItems = append(videoItems, item)
			}
		}
		if len(videoItems) > 0 {
			return videoItems, tag
		}
	}
	return []LibraryContentItem{}, selectedTags[len(selectedTags)-1]
}

func isVideoLibraryContent(item LibraryContentItem) bool {
	if item.ResourceType == AIEducationVideoType {
		return true
	}
	if item.Type == AIEducationVideoType {
		return true
	}
	for _, contentType := range item.ContentTypes {
		if contentType == AIEducationVideoType {
			return true
		}
	}
	return false
}

func parsePositiveInt(values url.Values, keys []string, defaultValue int) int {
	for _, key := range keys {
		value, err := strconv.Atoi(values.Get(key))
		if err == nil && value > 0 {
			return value
		}
	}
	return defaultValue
}

func isTruthy(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return raw == "1" || raw == "true" || raw == "yes" || raw == "all"
}

func sliceListPage(items []LibraryContentItem, values url.Values) []LibraryContentItem {
	if isTruthy(values.Get("all")) || isTruthy(values.Get("downloadAll")) {
		return items
	}

	pageSize := parsePositiveInt(values, []string{"pageSize", "page_size", "size", "limit"}, AIEducationDefaultPageSize)
	page := parsePositiveInt(values, []string{"page", "pageNum", "pageNo"}, 1)
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []LibraryContentItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func parseAIEducationListURL(parsedURL *url.URL, random bool) ([]string, error) {
	queryParams := parsedURL.Query()
	libraryID, selectedTags, err := parseAIEducationListParams(queryParams)
	if err != nil {
		return nil, err
	}

	server := pickServer(random)
	indexURL := buildLibraryAdapterIndexURL(server, libraryID)
	items, err := fetchAIEducationListItems(indexURL)
	if err != nil {
		return nil, err
	}

	videoItems, selectedTag := filterAIEducationVideoItems(items, selectedTags)
	if selectedTag != "" {
		slog.Info(fmt.Sprintf("AIEducation list selected tag %s", selectedTag))
	}
	videoItems = sliceListPage(videoItems, queryParams)

	configInfo := RESOURCES_MAP["/AIEducation/detail"]
	configURLList := []string{}
	for _, item := range videoItems {
		contentID := item.UnitID
		if contentID == "" {
			contentID = item.ID
		}
		if contentID == "" {
			continue
		}
		configURLList = append(configURLList, fmt.Sprintf(configInfo.resources.basic, server, contentID))
	}
	slog.Info(fmt.Sprintf("AIEducation list expanded to %d video resources", len(configURLList)))
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
			format = tiItem.TiFormat
			size = tiItem.TiSize
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
			if rawLink != "" {
				break
			}
		}

		fullTitle := concatFullTitle(title, bookName, schoolName, teacherNames)
		if rawLink != "" {
			backupLink := convertURL(rawLink, true) // 备用下载链接
			result = append(result, LinkData{
				Format:    format,
				Title:     fullTitle,
				ID:        item.ID,
				RawURL:    rawLink,
				BackupURL: backupLink,
				Size:      size,
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

func GenerateURLFromID2(linkItems []LinkItem) []string {
	urls := []string{}
	example_url := ReadingLibraryInfo.Detail
	for _, linkItem := range linkItems {
		url := fmt.Sprintf(example_url, linkItem.Link)
		urls = append(urls, url)
	}
	return urls
}
