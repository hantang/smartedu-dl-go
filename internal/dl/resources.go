package dl

const APP_DESC string = "本工具用于下载智慧教育平台中的教材等资源，支持批量下载PDF等资源。"
const LOG_FILE string = "log-smartedudl.txt"
const APP_NAME string = "cn.smartedu"

// 配置数据
// 服务器前缀
var SERVER_LIST = []string{
	"s-file-1",
	"s-file-2",
	"s-file-3",
}

// 下载数据格式（后缀）
var FORMAT_LIST = []FormatData{
	{"文档(PDF)", "pdf", true, true},
	{"音频(MP3)", "mp3", true, false},
	{"音频(OGG)", "ogg", true, false},
	{"图片", "jpg", true, false},
	// {"视频", "m3u8", false, false},
	{"白板", "whiteboard", true, false},
	{"字幕", "srt", true, false},
}

// folder

var FORMAT_VIDEO = []string{"m3u8"}

var TAB_NAMES = []string{
	"输入链接",
	"教材列表",
	"课件包",
	"诵读库",
}

var TAB_NAMES_LABEL = map[string][]string{
	"教材列表": {"🔍️ 点击查询、加载教材信息", "🗃️ 电子教材", "教材", "项"},
	"课件包":  {"🔍️ 点击加载课程教学内容", "⚗️ 课程单元", "课程", "项"},
	"诵读库":  {"🔍️ 点击加载诵读音频资料（语博书屋）", "🔊 课文音频", "诵读音频", "篇"},
}

// 电子教材（层级和列表数据等） https://basic.smartedu.cn/tchMaterial
var TchMaterialInfo = ResourceMetaInfo{
	Directory: "data/tchMaterial",
	Version:   "https://s-file-1.ykt.cbern.com.cn/zxx/ndrs/resources/tch_material/version/data_version.json",
	Tag:       "https://s-file-1.ykt.cbern.com.cn/zxx/ndrs/tags/tch_material_tag.json",
	Detail:    "https://basic.smartedu.cn/tchMaterial/detail?contentType=assets_document&contentId=%s",
	Type:      "tchMaterial",
}

// 课程教学>学生自主学习（课程包/课时：m3u8-视频，pdf-课件、教学设计、学习任务清单、课后练习） https://basic.smartedu.cn/syncClassroom
var SyncClassroomInfo = ResourceMetaInfo{
	Directory: "data/syncClassroom",
	Version:   "https://s-file-2.ykt.cbern.com.cn/zxx/ndrs/national_lesson/teachingmaterials/version/data_version.json",
	Tag:       "https://s-file-2.ykt.cbern.com.cn/zxx/ndrs/tags/national_lesson_tag.json",
	Detail:    "https://basic.smartedu.cn/syncClassroom/classActivity?activityId=%s",
	Type:      "national_lesson", // DataCourseInfo.ResourceType
}

// 语文诵读库 https://szyb.smartedu.cn/library/18cd1ab4-5560-472f-953a-85de88736aa2/catalog
var ReadingLibraryInfo = ResourceMetaInfo{
	Directory: "data/readingLibrary",
	Tag:       "https://s-file-1.ykt.cbern.com.cn/museum/api/zh-CN/a685009e-085c-4d18-9928-daa0ec5dd8a4/elearning_library/v1/libraries/18cd1ab4-5560-472f-953a-85de88736aa2/contents/actions/full/adapter/cd2d42991f0d3597fa8b5a5f5d2e5b5a1722e13d48d4d50db5b2e3c6c8ad9ae4.json",
	Detail:    "https://s-file-1.ykt.cbern.com.cn/museum/ndrs/special_edu/resources/details/%s.json",
	Type:      "assets_audio",
}

var EliteSyncClassroomInfo = ResourceMetaInfo{
	// 基础教育精品课
	Detail: "https://basic.smartedu.cn/qualityCourse?courseId=%s",
	Type:   "elite_lesson",
}

var InputInfo = ResourceMetaInfo{
	// 输入的链接，满足RESOURCES_MAP
	Detail: "",
	Type:   "",
}

// url path对应解析
var RESOURCES_MAP = map[string]ResourceData{
	"/tchMaterial/detail": {
		name:     "教材",
		params:   []string{"contentId"},
		examples: []string{},
		resources: ResourceInfo{
			// 课本PDF // 限制 contentType=assets_document
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/resources/tch_material/details/%s.json",
			// 备用 旧版本
			backup: []string{
				"https://%s.ykt.cbern.com.cn/zxx/ndrs/special_edu/resources/details/%s.json",
				"https://%s.ykt.cbern.com.cn/zxx/ndrs/resources/tch_material/details/%s.json",
				"https://%s.ykt.cbern.com.cn/zxx/ndrs/special_edu/thematic_course/%s/resources/list.json",
			},
			// 配套音频
			audio: "https://%s.ykt.cbern.com.cn/zxx/ndrs/resources/%s/relation_audios.json",
		},
		// 如果 contentType=thematic_course TODO
		// https://%s.ykt.cbern.com.cn/zxx/ndrs/special_edu/thematic_course/trees/%s.json 不一定有PDF
		// audio: https://s-file-2.ykt.cbern.com.cn/zxx/ndrs/resources/1bb3e2fe-45a1-2999-e8b4-9fc63d0929bb/relation_audios.json
	},

	"/syncClassroom/prepare/detail": {
		name:     "课程教学>教师授课备课>课件/教学设计",
		params:   []string{"resourceId"},
		examples: []string{},
		resources: ResourceInfo{
			// 课本、课件、视频等
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/prepare_sub_type/resources/details/%s.json",
		},
	},
	"/syncClassroom/classActivity": {
		// 学生自主学习 fromPrepare=1; 教师备课资源  fromPrepare=0
		name:     "课程教学>学生自主学习, 课程教学>教师备课资源>课程包", // 国家课
		params:   []string{"activityId"},
		examples: []string{},
		resources: ResourceInfo{
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/national_lesson/resources/details/%s.json",
		},
	},
	"/qualityCourse": {
		name:   "课程教学>学生自主学习(基础教育精品课程)", // 精品课
		params: []string{"courseId"},
		examples: []string{
			// url需要chapterId才能打开
			"https://basic.smartedu.cn/qualityCourse?courseId=78605dce-97f6-62b7-6cb2-ac41ffd9467b&chapterId=1b2575b9-16f6-3501-8bbf-2d259fea97cf",
		},
		resources: ResourceInfo{
			// https://s-file-1.ykt.cbern.com.cn/zxx/ndrv2/resources/78605dce-97f6-62b7-6cb2-ac41ffd9467b.json
			basic: "https://%s.ykt.cbern.com.cn/zxx/ndrv2/resources/%s.json",
		},
	},
	// "/syncClassroom/examinationpapers": {
	// 	name:     "课程教学>教师授课备课>习题资源", // 数据是json 忽略
	// 	params:   []string{"resourceId"},
	// 	examples: []string{
	// 		"https://basic.smartedu.cn/syncClassroom/examinationpapers?resourceId=95af8600-c178-488e-98ce-918106d4fdba&chapterId=538ac938-a87d-37e9-9a3c-a2fb8322329e&teachingmaterialId=d92ca54e-2cdc-4921-95f3-769eafd0c814&fromPrepare=1",
	// 	},
	// 	resources: ResourceInfo{
	// 		basic: "https://%s.ykt.cbern.com.cn/zxx/ndrs/examinationpapers/resources/details/%s.json", // -> create_container_id
	// step2: "https://bdcs-file-2.ykt.cbern.com.cn/xedu_cs_paper_bank/api_static/papers/${create_container_id}/data.json" // -> question_path_list
	// step3: "https://bdcs-file-2.ykt.cbern.com.cn/xedu_cs_paper_bank/api_static/papers/${question_path_list[0]}/question_files/0.json"
	// 	},
	// },
}

var RESOURCES_PATH = "/edu_product/esp/assets/"

// 数据结构
type ResourceMetaInfo struct {
	Directory string
	Version   string
	Tag       string
	Detail    string
	Type      string
}

type ResourceInfo struct {
	basic  string
	backup []string
	audio  string
}

type ResourceData struct {
	name      string
	params    []string
	examples  []string
	resources ResourceInfo
}

// 资源文件中抽取得到格式（后缀）、标题（文件名）和下载链接
type LinkData struct {
	Format    string
	Title     string
	ID        string
	RawURL    string
	BackupURL string
	Size      int64
}

type FormatData struct {
	Name   string
	Suffix string
	Status bool
	Check  bool
}

// 教材PDF信息
type DocPDFData struct {
	ID      string
	Title   string
	TagPath string
	TagID   string
}

// 教材层级结构
type BookItem struct {
	Level    int
	Name     string // hierarchy_name
	TagID    string
	TagName  string
	BookID   string
	BookName string
	IsBook   bool
	Children []BookItem
}

// 下拉框/多选框
type BookOption struct {
	OptionID   string
	OptionName string
}

// JSON数据解析
// 资源所在JSON的格式
type ResourceItemExt struct {
	ID    string `json:"id"`
	Title struct {
		Name string `json:"zh-CN"`
	} `json:"global_title"`

	Relations struct {
		NationalCourseResource []ResourceItem `json:"national_course_resource"`
		EliteCourseResource    []ResourceItem `json:"course_resource"`
	} `json:"relations"`
	TeacherList      []Teacher `json:"teacher_list"`
	CustomProperties struct {
		SchoolName string   `json:"school_name"`
		BookInfo   BookInfo `json:"teachingmaterial_info"`
	} `json:"custom_properties"`
}

type Teacher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 资源文件
type ResourceItem struct {
	ID               string   `json:"id"`
	TiItems          []TiItem `json:"ti_items"`
	Title            string   `json:"title"`
	ResourceType     string   `json:"resource_type_code_name"`
	CustomProperties struct {
		OriginalTitle string `json:"original_title"`
		AliasName     string `json:"alias_name"`
	} `json:"custom_properties"`
}

type BookInfo struct {
	Name         string   `json:"title"`
	ID           string   `json:"id"`
	ResourceType string   `json:"resource_type_code_name"`
	Language     string   `json:"language"`
	TagList      []DocTag `json:"tag_list"`
}

// 资源文件中ti_items
type TiItem struct {
	TiStorages       []string `json:"ti_storages"`
	TiFormat         string   `json:"ti_format"`
	LcTiFormat       string   `json:"lc_ti_format"`
	TiSize           int64    `json:"ti_size"`
	TiFileFlag       string   `json:"ti_file_flag"`
	TiIsSourceFile   bool     `json:"ti_is_source_file"`
	CustomProperties struct {
		Encryption     string      `json:"encryption"`
		Identification bool        `json:"identification"`
		Requirements   []SubTiItem `json:"requirements"`
	} `json:"custom_properties"`
}

type SubTiItem struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// tch_material_tag.json 结构
type TagExt struct {
	// TagDimensionID string   `json:"tag_dimension_id"`
	HasNextTagPath []string `json:"has_next_tag_path"`
	HiddenTags     []string `json:"hidden_tags"`
}

type TagHierarchy struct {
	Children      []TagItem `json:"children"`
	Ext           *TagExt   `json:"ext"`
	HierarchyName string    `json:"hierarchy_name"`
}

type TagItem struct {
	TagID       string         `json:"tag_id"`
	TagName     string         `json:"tag_name"`
	Hierarchies []TagHierarchy `json:"hierarchies"`
	Ext         interface{}    `json:"ext"`
}

type TagBase struct {
	TagID       string         `json:"tag_path"`
	Hierarchies []TagHierarchy `json:"hierarchies"`
	Ext         interface{}    `json:"ext"`
}

// part_100.json 数据
type DocTag struct {
	TagID    string `json:"tag_id"`
	TagName  string `json:"tag_name"`
	TagDim   string `json:"tag_dimension_id"`
	OrderNum int    `json:"order_num"`
}

type DocResourceItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	ResourceType string   `json:"resource_type_code"`
	TagPaths     []string `json:"tag_paths"`
	TagList      []DocTag `json:"tag_list"`
}

// data_version.json
type DataVersion struct {
	Module        string      `json:"module"`
	ModuleVersion int64       `json:"module_version"`
	URLs          interface{} `json:"urls"` // can be string or []string
}

// national_lesson 课程
type DataCourseInfo struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	TeachIDs         []string `json:"teachmeterial_ids"`
	ChapterPaths     []string `json:"chapter_paths"`           // 只取第1个 和 NodePath关联
	ResourceType     string   `json:"resource_type_code"`      // national_lesson / elite_lesson
	ResourceTypeName string   `json:"resource_type_code_name"` // 国家课 / 精品课
	// 更多字段 relations
}

// examinationpapers / 试卷
// national_lesson / 国家课
// elite_lesson / 精品课

// 课程目录
type DataCourseChapter struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	NodePath   string              `json:"node_path"`
	TreeID     string              `json:"tree_id"`
	CreateTime string              `json:"create_time"`
	UpdateTime string              `json:"update_time"`
	Children   []DataCourseChapter `json:"child_nodes"`
}

type CourseItem struct {
	Title        string
	NodePath     string // DataCourseChapter.NodePath
	NodeID       string // DataCourseChapter.ID
	NodeTitle    string // DataCourseChapter.Title
	NodeParents  []string
	CourseID     string // DataCourseInfo.ID
	CourseTitle  string
	ResourceType string
}

type CourseToc struct {
	Index    int
	Title    string
	Children []CourseItem
}

type LinkItem struct {
	Link string
	Type string // ResourceMetaInfo.Type
}

// readinglibrary
type DataLibrary struct {
	Files     []string `json:"files"`
	Timestamp int64    `json:"timestamp"`
}

type ReadingItem struct {
	UnitID       string       `json:"unit_id"`
	ResourceType string       `json:"resource_type"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Tags         []ReadingTag `json:"tags"`
}

type ReadingTag struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"`
}
