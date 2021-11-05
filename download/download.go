package download

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/apex/log"
	"github.com/bytedance/sonic"
	"github.com/rammiah/bili-downloader/download/httpcli"
)

type DownloadInfo struct {
	VideoID string `json:"video_id"`
	Avid    int64  `json:"avid"`
	Cid     int64  `json:"cid"`
	Qn      int64  `json:"qn"`
	Length  int64  `json:"length"`
	Size    int64  `json:"size"`
	Url     string `json:"url"`
}

func GetDownloadInfoByAidCid(videoId string, avid, cid int64) (*DownloadInfo, error) {
	const (
		QUrl = "https://api.bilibili.com/x/player/playurl"
	)

	req, err := http.NewRequest(http.MethodGet, QUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("user-agent", httpcli.UA)
	params := map[string]string{
		"avid":  strconv.FormatInt(avid, 10),
		"cid":   strconv.FormatInt(cid, 10),
		"otype": "json",
		"qn":    "125", // 画质直接按照最高获取
		"fourk": "1",
		"fnver": "0",
		"fnval": "0",
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}

	log.Debugf("query params: %v", q.Encode())
	req.URL.RawQuery = q.Encode()

	resp, err := httpcli.Inst.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Errorf("status code invalid: %v", resp.StatusCode)
		return nil, fmt.Errorf("invalid status code %v", resp.StatusCode)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if code, err := sonic.Get(buf, "code"); err != nil {
		return nil, err
	} else if code, err := code.Int64(); err != nil {
		return nil, err
	} else if code != 0 {
		return nil, fmt.Errorf("ret code not 0: %v", code)
	}

	data, err := sonic.Get(buf, "data")
	if err != nil {
		return nil, err
	}
	durl := data.Get("durl")
	if err := durl.Load(); err != nil {
		return nil, err
	}

	if n, err := durl.Len(); err != nil {
		return nil, err
	} else if n == 0 {
		return nil, fmt.Errorf("not valid durl, count is 0")
	}

	obj := durl.Index(0)
	var (
		length, _ = obj.Get("length").Int64()
		size, _   = obj.Get("size").Int64()
		u, _      = obj.Get("url").String()
		qn, _     = data.Get("quality").Int64()
	)

	return &DownloadInfo{
		VideoID: videoId,
		Avid:    avid,
		Cid:     cid,
		Qn:      qn,
		Length:  length,
		Size:    size,
		Url:     u,
	}, nil
}

func DownloadVideo(info *DownloadInfo, target io.Writer) error {
	/*
		accept: *\*
			accept-encoding: identity
			accept-language: zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7
			dnt: 1
			origin: https://www.bilibili.com
			range: bytes=218103821-234881037
			referer: https://www.bilibili.com/video/BV1gF411Y7Z3
			sec-ch-ua: "Google Chrome";v="95", "Chromium";v="95", ";Not A Brand";v="99"
			sec-ch-ua-mobile: ?0
			sec-ch-ua-platform: "Windows"
			sec-fetch-dest: empty
			sec-fetch-mode: cors
			sec-fetch-site: cross-site
			user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36
	*/
	// u, err := url.Parse(info.Url)
	// if err != nil {
	//     return err
	// }
	// fmt.Printf("url is %v\n", utils.Json(u))
	// q, _ := url.ParseQuery(u.RawQuery)
	// fmt.Printf("query is %v\n", utils.Json(q))
	if err := AuthVideo(info.VideoID, info.Url); err != nil {
		log.Errorf("auth video error: %v", err)
		return err
	}

	req, err := http.NewRequest(http.MethodGet, info.Url, nil)
	if err != nil {
		return err
	}
	params := map[string]string{
		"accept":             "*/*",
		"accept-encoding":    "identity",
		"accept-language":    "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
		"dnt":                "1",
		"origin":             "https://www.bilibili.com",
		"referer":            "https://www.bilibili.com/video/" + info.VideoID,
		"sec-ch-ua":          `"Google Chrome";v="95", "Chromium";v="95", ";Not A Brand";v="99"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "Windows",
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "cross-site",
		"user-agent":         httpcli.UA,
	}

	for k, v := range params {
		req.Header.Set(k, v)
	}

	// set range header
	req.Header.Set("range", "0-"+strconv.FormatInt(info.Size-1, 10))

	resp, err := httpcli.Inst.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get status not ok: %v", resp.Status)
	}
	if resp.ContentLength != info.Size {
		return fmt.Errorf("content size not same: expect %v got %v", info.Size, resp.ContentLength)
	}

	cnt, err := io.Copy(target, resp.Body)
	if err != nil {
		return err
	}

	log.Infof("download size %v bytes", cnt)

	return nil
}

func AuthVideo(videoId, videoUrl string) error {
	req, err := http.NewRequest(http.MethodOptions, videoUrl, nil)
	if err != nil {
		log.Errorf("new request error: %v", err)
		return err
	}
	head := map[string]string{
		// ":authority":                     req.URL.Host,
		// ":method":                        http.MethodOptions,
		// ":path":                          req.URL.Path,
		// ":scheme":                        req.URL.Scheme,
		"accept":                         "*/*",
		"accept-encoding":                "gzip, deflate, br",
		"access-control-request-headers": "range",
		"access-control-request-method":  http.MethodGet,
		"origin":                         "https://www.bilibili.com",
		"referer":                        "https://www.bilibili.com/" + videoId,
		"sec-fetch-dest":                 "empty",
		"sec-fetch-mode":                 "cors",
		"sec-fetch-site":                 "cross-site",
		"user-agent":                     httpcli.UA,
	}

	for k, v := range head {
		req.Header.Set(k, v)
	}

	resp, err := httpcli.Inst.Do(req)
	if err != nil {
		log.Errorf("do request error: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Errorf("request code not ok", resp.StatusCode)
		return fmt.Errorf("options request status code not ok %v", resp.StatusCode)
	}

	return nil
}
