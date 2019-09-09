package model

import (
	"fmt"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

// 过滤处理函数
type EncoderMap map[string]jsoniter.ValEncoder
type Encoder jsoniter.ValEncoder

func (fm EncoderMap) AddEncoder(name string, fun Encoder) {
	fm[name] = fun
}

func LoadEncoderMap() EncoderMap {
	fm := make(EncoderMap)

	// add more build-in function
	fm.AddEncoder("remove", &omitEncoder{})         // 移除字段
	fm.AddEncoder("sensitive", &sensitiveEncoder{}) // 脱敏

	return fm
}

// 移除字段
type omitEncoder struct{}

func (encoder *omitEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
}
func (encoder *omitEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	return true
}
func (encoder *omitEncoder) IsEmbeddedPtrNil(ptr unsafe.Pointer) bool {
	return true
}

// 脱敏
type sensitiveEncoder struct{}

func (encoder *sensitiveEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	str := *((*string)(ptr))
	strlen := len(str)
	newstr := ""
	if strlen <= 4 { // 1234 => 1****4
		newstr = fmt.Sprintf("%s****%s", str[0:1], str[strlen-1:])
	} else { // 123456 => 1234****3456
		newstr = fmt.Sprintf("%s****%s", str[0:4], str[strlen-4:])
	}
	stream.WriteString(newstr)
}
func (encoder *sensitiveEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	return *((*string)(ptr)) == ""
}
