package util

import (
	"sort"
	"strings"
)

var cnNums = map[string]int{
	"一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6,
	"七": 7, "八": 8, "九": 9, "十": 10, "十一": 11, "十二": 12,
}

type Grade struct {
	Raw   string
	Stage int // 小学=1 初中=2 高中=3
	Num   int // 年级数字
	Term  int // 0=无, 1=上, 2=下
}

func parseGrade(orig string) Grade {
	s := strings.ReplaceAll(orig, " ", "")
	stage := 1
	if strings.HasPrefix(s, "初") {
		stage = 2
		s = strings.TrimPrefix(s, "初")
	} else if strings.HasPrefix(s, "高") {
		stage = 3
		s = strings.TrimPrefix(s, "高")
	}

	// 优先去掉学期后缀（上/下册）
	term := 0
	if strings.HasSuffix(s, "上册") {
		term = 1
		s = strings.TrimSuffix(s, "上册")
	} else if strings.HasSuffix(s, "下册") {
		term = 2
		s = strings.TrimSuffix(s, "下册")
	}

	// 去掉常见后缀
	for {
		switch {
		case strings.HasSuffix(s, "年级"):
			s = strings.TrimSuffix(s, "年级")
		case strings.HasSuffix(s, "年"):
			s = strings.TrimSuffix(s, "年")
		case strings.HasSuffix(s, "级"):
			s = strings.TrimSuffix(s, "级")
		default:
			goto DONE
		}
	}
DONE:

	num := 0
	if v, ok := cnNums[s]; ok {
		num = v
	}
	return Grade{Raw: orig, Stage: stage, Num: num, Term: term}
}

func SortGrades(list []string) []string {
	grades := make([]Grade, len(list))
	for i, s := range list {
		grades[i] = parseGrade(s)
	}
	sort.SliceStable(grades, func(i, j int) bool {
		if grades[i].Stage != grades[j].Stage {
			return grades[i].Stage < grades[j].Stage
		}
		if grades[i].Num != grades[j].Num {
			return grades[i].Num < grades[j].Num
		}
		return grades[i].Term < grades[j].Term
	})

	out := make([]string, len(grades))
	for i, g := range grades {
		out[i] = g.Raw
	}
	return out
}
