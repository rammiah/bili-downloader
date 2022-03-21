package consts

import "fmt"

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * MB
	PB = 1024 * MB
)

const (
	FragSize = 8 * MB
)

var FormatBiliToFile = map[string]string{
	"hdflv2":  "flv",
	"flv_p60": "flv",
	"flv":     "flv",
	"flv720":  "flv",
	"flv480":  "flv",
	"mp4":     "mp4",
}

type Byte int64

func (b Byte) String() string {
	if b < 0 {
		return "NA"
	}
	var result string
	switch {
	case b < KB:
		result = fmt.Sprintf("%d B", b)
	case b < MB:
		result = fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	case b < GB:
		result = fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b < TB:
		result = fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b < PB:
		result = fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	default:
		result = fmt.Sprintf("%.2f PB", float64(b)/float64(PB))
	}
	return result
}
