package internal

type ResourceInfo struct {
	basic  string
	backup string
	audio  string
}

type ResourceData struct {
	name      string
	params    []string
	examples  []string
	resources ResourceInfo
}

// 顶层响应结构
type ResourceResponse struct {
	Relations struct {
		NationalCourseResource []ResourceItem `json:"national_course_resource"`
	} `json:"relations"`
}

// 资源文件
type ResourceItem struct {
	TiItems      []TiItem `json:"ti_items"`
	Title        string   `json:"title"`
	ResourceType string   `json:"resource_type_code_name"`
}

// 资源文件中ti_items
type TiItem struct {
	TiStorages []string `json:"ti_storages"`
	TiFormat   string   `json:"ti_format"`
	TiSize     int64    `json:"ti_size"`
}

// 资源文件中抽取得到格式（后缀）、标题（文件名）和下载链接
type LinkData struct {
	format string
	title  string
	url    string
	size   int64
}

// 服务器前缀
var SERVER_LIST = []string{
	"s-file-1",
	"s-file-2",
	"s-file-3",
}

// url path对应解析
var RESOURCES_MAP = map[string]ResourceData{
	"/tchMaterial/detail": {
		name:     "教材",
		params:   []string{"contentId"},
		examples: []string{},
		resources: ResourceInfo{
			// 课本PDF
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/resources/tch_material/details/%s.json",
			// 备用 旧版本
			backup: "https://%s.ykt.cbern.com.cn/zxx/ndrs/resources/tch_material/details/%s.json",
			// 配套音频
			audio: "https://%s.ykt.cbern.com.cn/zxx/ndrs/resources/%s/relation_audios.json",
		},
	},

	"/syncClassroom/prepare/detail": {
		name:     "课程教学>教师授课备课",
		params:   []string{"resourceId"},
		examples: []string{},
		resources: ResourceInfo{
			// 课本、课件、视频等
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/prepare_sub_type/resources/details/%s.json",
		},
	},
	"/syncClassroom/classActivity": {
		name:     "课程教学>学生自主学习, 课程教学>教师备课资源",
		params:   []string{"activityId"},
		examples: []string{},
		resources: ResourceInfo{
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/national_lesson/resources/details/%s.json",
		},
	},
	// TODO 
	// https://basic.smartedu.cn/syncClassroom/examinationpapers
}

type FormatData struct {
	name   string
	suffix string
	status bool
	check  bool
}

var FORMAT_LIST = []FormatData{
	{"文档(PDF)", "pdf", true, true},
	{"音频(MP3)", "mp3", true, false},
	{"音频(OGG)", "ogg", true, false},
	{"图片", "jpg", true, false},
	{"视频", "m3u8", false, false},
	{"白板", "whiteboard", true, false},
}

const APP_NAME string = "smartedu-dl"
const APP_DESC string = "本工具用于下载智慧教育平台中的教材资源，支持批量下载PDF等资源。"
const APP_VERSION string = "0.0.1"
