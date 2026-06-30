package util

import (
	"encoding/json"
	"fmt"
)

type StringOrList []string

func (s *StringOrList) UnmarshalJSON(data []byte) error {
	// string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrList{str}
		return nil
	}

	// []string
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = StringOrList(arr)
		return nil
	}

	return fmt.Errorf("invalid StringOrList: %s", string(data))
}

// 第一个值
func (s StringOrList) First() string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
