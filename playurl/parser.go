package playurl

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/apex/log"
	"github.com/bytedance/sonic"
)

const (
	kVideoUrl = "https://www.bilibili.com/video/"
	kUrlUrl   = "https://api.bilibili.com/x/player/playurl"
)

var (
	ErrUnrecognizedId = errors.New("unrecognized id type")
	ErrNetRequest     = errors.New("make http request error")
)
var (
	//go:embed cookie.txt
	cookie string
)

func init() {
	cookie = strings.TrimSpace(cookie)
}

type PlayUrl struct {
	Aid      int64  `json:"aid"`
	Cid      int64  `json:"cid"`
	Page     int64  `json:"page"`     // page no
	Duration int64  `json:"duration"` // length in seconds
	Part     string `json:"part"`     // part name
}

func GetPlayUrlsById(id string) ([]*PlayUrl, error) {
	if len(id) < 2 {
		return nil, ErrUnrecognizedId
	}
	switch id[:2] {
	case "av", "BV":
	// done
	default:
		return nil, ErrUnrecognizedId
	}

	req, err := http.NewRequest(http.MethodGet, kVideoUrl+id, nil)
	if err != nil {
		log.WithError(err).Error("http new request error")
		return nil, err
	}

	if cookie != "" {
		req.Header.Add("Cookie", cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithError(err).Error("do http request error")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code not ok: %v", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.WithError(err).Error("parse html error")
		return nil, err
	}
	sel := doc.Find("script")

	var (
		playUrls []*PlayUrl
	)

	for _, node := range sel.Nodes {
		const (
			JS_START = "window.__INITIAL_STATE__="
			JS_END   = ";(function(){var s;"
		)
		if child := node.FirstChild; child != nil {
			if !strings.Contains(child.Data, JS_START) {
				continue
			}
			start := strings.Index(child.Data, JS_START)
			end := strings.LastIndex(child.Data, JS_END)
			if start == -1 || end == -1 {
				continue
			}
			start += len(JS_START)
			jsTxt := child.Data[start:end]
			aidNode, err := sonic.GetFromString(jsTxt, "aid")
			if err != nil {
				return nil, err
			}
			var aid int64
			if v, err := aidNode.Int64(); err != nil {
				return nil, err
			} else {
				aid = v
			}
			pages, err := sonic.GetFromString(jsTxt, "videoData", "pages")
			if err != nil {
				return nil, err
			}
			// load childen nodes
			_ = pages.Load()
			cnt, err := pages.Len()
			if err != nil {
				return nil, err
			}

			for i := 0; i < cnt; i++ {
				page := pages.Index(i)
				cid, _ := page.Get("cid").Int64()
				pageNo, _ := page.Get("page").Int64()
				part, _ := page.Get("part").String()
				length, _ := page.Get("duration").Int64()
				url := &PlayUrl{
					Aid:      aid,
					Cid:      cid,
					Page:     pageNo,
					Duration: length,
					Part:     part,
				}
				playUrls = append(playUrls, url)
			}
			log.Infof("parse url for %v success, aid %v, cids count %v", id, aid, len(playUrls))
			// parse is over
			break
		}
	}

	return playUrls, nil
}
