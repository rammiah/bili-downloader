package download

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/download/httpcli"
	"github.com/tidwall/gjson"
)

const (
	kVideoUrl = "https://www.bilibili.com/video/"
	kUrlUrl   = "https://api.bilibili.com/x/player/playurl"
)

type VideoInfo struct {
	VideoID  string `json:"video_id"` // video id, av/BV
	Avid     int64  `json:"avid"`
	Cid      int64  `json:"cid"`
	Page     int64  `json:"page"`     // page no
	Duration int64  `json:"duration"` // length in seconds
	Part     string `json:"part"`     // part name
}

type UrlProcessor struct {
	videoId string
	urls    []*VideoInfo
}

// GetVideoInfosById get videos id infomation by id
func GetVideoInfosById(id string) ([]*VideoInfo, error) {
	p := &UrlProcessor{
		videoId: id,
	}

	if err := p.CheckArgs(); err != nil {
		log.Errorf("check args error: %v", err)
		return nil, err
	}

	if err := p.QueryAidCids(); err != nil {
		log.Errorf("query aid and cid error: %v", err)
		return nil, err
	}

	return p.urls, nil
}

// CheckArgs check if video id legal
func (p *UrlProcessor) CheckArgs() error {
	if len(p.videoId) < 2 {
		return fmt.Errorf("id length too short: %v", len(p.videoId))
	}

	switch p.videoId[:2] {
	case "av", "BV":
		// donothing
	default:
		return fmt.Errorf("unrecognized video id %v, should starts with av/BV", p.videoId)
	}

	log.Infof("check video id %v passed", p.videoId)

	return nil
}

// QueryAidCids get every clip's aid, cid
func (p *UrlProcessor) QueryAidCids() error {
	req, err := http.NewRequest(http.MethodGet, kVideoUrl+p.videoId, nil)
	if err != nil {
		log.WithError(err).Error("http new request error")
		return err
	}

	resp, err := httpcli.Inst.Do(req)
	if err != nil {
		log.WithError(err).Error("do http request error")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not ok: %v", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.WithError(err).Error("parse html error")
		return err
	}
	sel := doc.Find("script")

	var (
		playUrls []*VideoInfo
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
			if !gjson.Valid(jsTxt) {
				log.Infof("js text is valid json")
				return errors.New("invalid json detected")
			}
			avid := gjson.Get(jsTxt, "aid").Int()
			pages := gjson.Get(jsTxt, "videoData.pages")
			if !pages.Exists() {
				return errors.New("pages not exists")
			}
			if !pages.IsArray() {
				return errors.New("pages not array")
			}

			for _, page := range pages.Array() {
				cid := page.Get("cid").Int()
				pageNo := page.Get("page").Int()
				part := page.Get("part").String()
				length := page.Get("duration").Int()
				url := &VideoInfo{
					VideoID:  p.videoId,
					Avid:     avid,
					Cid:      cid,
					Page:     pageNo,
					Duration: length,
					Part:     part,
				}
				// buf, _ := page.MarshalJSON()
				// log.Infof("page content is %s", buf)
				log.Infof("parse page %v, part %v success", pageNo, part)
				playUrls = append(playUrls, url)
			}
			if len(playUrls) == 1 {
				title := gjson.Get(jsTxt, "videoData.title").String()
				log.Infof("only 1 video, use title %v for part name", title)
				playUrls[0].Part = title
			}
			log.Infof("parse url for %v success, aid %v, cids count %v", p.videoId, avid, len(playUrls))
			p.urls = playUrls
			// parse is over
			break
		}
	}

	return nil
}
