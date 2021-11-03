package utils

import (
	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func Json(obj interface{}) string {
	buf, err := json.Marshal(obj)
	if err != nil {
		return "<error>"
	}
	return string(buf)

}
