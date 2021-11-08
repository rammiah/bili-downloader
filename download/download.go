package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/consts"
	"github.com/rammiah/bili-downloader/download/httpcli"
	"github.com/tidwall/gjson"
)

type DownloadInfo struct {
	VideoID string `json:"video_id"`
	Avid    int64  `json:"avid"`
	Cid     int64  `json:"cid"`
	Qn      int64  `json:"qn"`
	Length  int64  `json:"length"`
	Size    int64  `json:"size"`
	Url     string `json:"url"`
	Format  string `json:"format"`
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

	code := gjson.GetBytes(buf, "code")
	if code.Type == gjson.Null {
		return nil, errors.New("null json")
	} else if code.Int() != 0 {
		return nil, fmt.Errorf("code not 0: %v", code.Int())
	}

	data := gjson.GetBytes(buf, "data")
	durl := data.Get("durl")

	if len(durl.Array()) == 0 {
		return nil, fmt.Errorf("not valid durl, count is 0")
	}

	obj := durl.Get("0")
	var (
		length = obj.Get("length").Int()
		size   = obj.Get("size").Int()
		u      = obj.Get("url").String()
		qn     = data.Get("quality").Int()
		format = data.Get("format").String()
	)
	if v, ok := consts.FormatBiliToFile[format]; ok {
		format = v
	}

	return &DownloadInfo{
		VideoID: videoId,
		Avid:    avid,
		Cid:     cid,
		Qn:      qn,
		Length:  length,
		Size:    size,
		Url:     u,
		Format:  format,
	}, nil
}

func authVideo(videoId, videoUrl string) error {
	req, err := http.NewRequest(http.MethodOptions, videoUrl, nil)
	if err != nil {
		log.Errorf("new request error: %v", err)
		return err
	}
	head := map[string]string{
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
