package consts

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

const (
	FragSize = 4 * MB
)

var FormatBiliToFile = map[string]string{
	"hdflv2":  "flv",
	"flv_p60": "flv",
	"flv":     "flv",
	"flv720":  "flv",
	"flv480":  "flv",
	"mp4":     "mp4",
}
