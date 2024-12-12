package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
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
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	// 提取主机、路径和查询参数
	// host := parsedURL.Host
	path := parsedURL.Path
	queryParams := parsedURL.Query()
	configInfo, ok := RESOURCES_MAP[path]
	if !ok {
		return nil, fmt.Errorf("invalid map key: %s", path)
	}

	contentTypeKey := "contentType"
	serverKey := "server"
	var paramList []string = append(configInfo.params, contentTypeKey)
	paramDict := map[string]string{}
	for _, key := range paramList {
		paramDict[key] = queryParams.Get(key)
	}
	if random {
		paramDict[serverKey] = SERVER_LIST[rand.Intn(len(SERVER_LIST))]
	} else {
		paramDict[serverKey] = SERVER_LIST[0]
	}

	var configURLList []string
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

	// 可能是数组或者字典对象
	var items []ResourceItem
	if err := json.Unmarshal(data, &items); err != nil {
		var item ResourceItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		items = []ResourceItem{item}
	}

	for i, item := range items {
		var title = item.Title
		if title == "" && item.ResourceType != "" {
			title = fmt.Sprintf("%s-%03d", item.ResourceType, i)
		}
		var link string
		var format string
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
			format = tiItem.TiFormat
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
				format: format,
				title:  title,
				url:    link,
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
		data, err := FetchJsonData(url)
		if err != nil {
			continue
		}
		resources, err := parseResourceItems(data, formatList, random)
		if err != nil {
			continue
		}
		result = append(result, resources...)
	}
	return result
}

