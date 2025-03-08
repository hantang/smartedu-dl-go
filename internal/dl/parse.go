package dl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"regexp"
	"slices"
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
	if _, ok := RESOURCES_MAP[path]; !ok {
		return false
	}
	return true
}

func parseURL(link string, audio bool, random bool) ([]string, error) {
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

	if audio && configInfo.resources.audio != "" {
		audioURL := fmt.Sprintf(configInfo.resources.audio, paramDict[serverKey], paramDict[configInfo.params[0]])
		configURLList = append(configURLList, audioURL)
	}
	return configURLList, nil
}

func parseURLList(links []string, audio bool, random bool) []string {
	var configURLList []string
	for _, link := range links {
		output, err := parseURL(link, audio, random)
		if err != nil {
			slog.Debug(fmt.Sprintf("parse link error: %v", err))
			continue
		}
		configURLList = append(configURLList, output...)
	}
	return configURLList
}

func parseResourceItems(data []byte, tiFormatList []string, random bool) ([]LinkData, error) {
	// 解析资源文件（json格式），得到最终下来文件的链接
	var result []LinkData
	var items []ResourceItem

	// 尝试解析为ResourceItemExt
	var response ResourceItemExt
	if err := json.Unmarshal(data, &response); err == nil {
		if len(response.Relations.NationalCourseResource) > 0 {
			items = response.Relations.NationalCourseResource
		}
	}

	// 如果不是ResourceItemExt，尝试解析为ResourceItem数组
	if len(items) == 0 {
		if err := json.Unmarshal(data, &items); err != nil {
			// 如果不是数组，尝试解析为单个ResourceItem
			var item ResourceItem
			if err := json.Unmarshal(data, &item); err != nil {
				return nil, err
			}
			items = []ResourceItem{item}
		}
	}

	// 处理每个ResourceItem
	for i, item := range items {
		title := item.Title
		if title == "" && item.ResourceType != "" {
			title = fmt.Sprintf("%s-%03d", item.ResourceType, i)
		}
		var link string
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
			link = tiItem.TiStorages[randomIndex]
			// link ~~去除“-doc-private”~~
			// => https://r3-ndr.ykt.cbern.com.cn/edu_product/esp/assets/<教材代码>.pkg/pdf.pdf
			// var materialId = link.split("");
			re := regexp.MustCompile(`([a-z\d\-]+).pkg`)
			materialId := re.FindString(link)
			slog.Debug(fmt.Sprintf("URL = %s / materialId=%s", link, materialId))
			if len(materialId) == 0 {
				continue
			}
			link = fmt.Sprintf("https://r%d-ndr.ykt.cbern.com.cn/edu_product/esp/assets/%s/pdf.pdf", rune(randomIndex)+1, materialId)
			slog.Debug(fmt.Sprintf("Update URL = %s", link))

			format = tiItem.TiFormat
			size = tiItem.TiSize
			if title == "" {
				title = fmt.Sprintf("%s-%03d", strings.ToUpper(format), i)
			}
			if link != "" {
				break
			}
		}
		if link != "" {
			link = cleanURL(link)
			result = append(result, LinkData{
				Format: format,
				Title:  title,
				URL:    link,
				Size:   size,
			})
		}
	}

	return result, nil
}

func ExtractResources(links []string, formatList []string, random bool) []LinkData {
	var result []LinkData

	var audio = false
	for _, format := range formatList {
		if format == "mp3" || format == "ogg" {
			audio = true
			break
		}
	}
	slog.Debug(fmt.Sprintf("formats = %v, audio is %v", formatList, audio))

	configURLList := parseURLList(links, audio, random)
	slog.Debug(fmt.Sprintf("configURLList is %v", len(configURLList)))

	for _, url := range configURLList {
		slog.Debug("config url = " + url)
		data, err := FetchJsonData(url)
		if err != nil {
			slog.Warn(fmt.Sprintf("fetch data error: %v", err))
			continue
		}
		resources, err := parseResourceItems(data, formatList, random)
		if err != nil {
			slog.Warn(fmt.Sprintf("parse resource data error: %v", err))
			continue
		}
		result = append(result, resources...)
	}
	return result
}
