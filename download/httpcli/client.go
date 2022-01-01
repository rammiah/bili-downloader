package httpcli

import (
	"net/http"
	"time"

	"github.com/rammiah/bili-downloader/download/cookie"
)

const (
	UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36"
)

var (
	Inst *http.Client
)

func init() {
	Inst = &http.Client{
		Jar:     cookie.GetCookieJar(),
		Timeout: time.Minute,
	}
}
