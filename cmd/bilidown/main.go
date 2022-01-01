package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/download"
	"github.com/rammiah/bili-downloader/download/cookie"
)

func main() {
	defer cookie.SaveCookies()
	var (
		id      string
		pageStr string
	)
	flag.StringVar(&id, "id", "", "video id like avxxx/BVxxx")
	flag.StringVar(&pageStr, "p", "", "page to download")
	flag.Parse()
	id = strings.TrimSpace(id)
	if id == "" {
		flag.Usage()
		os.Exit(-1)
	}

	pageMatch, err := parsePages(pageStr)
	if err != nil {
		log.Errorf("parse page matcher error: %v", err)
		return
	}

	infos, err := download.GetVideoInfosById(id)
	if err != nil {
		panic(err)
	}
	// filter video infos
	idx := 0
	for _, info := range infos {
		if pageMatch(info.Page) {
			infos[idx] = info
			idx++
		}
	}
	infos = infos[:idx]
	// fmt.Printf("%v\n", utils.Json(resp))
	for _, video := range infos {
		log.Infof("process avid %v, cid %v", video.Avid, video.Cid)
		info, err := download.GetDownloadInfoByAidCid(id, video.Avid, video.Cid)
		if err != nil {
			panic(err)
		}
		fileName := video.PartName + "." + info.Format
		if len(infos) > 1 {
			fileName = video.Title + " - " + fileName
		}
		fileName = strings.ReplaceAll(fileName, "/", " ")
		log.Infof("start download file %v, size %v bytes", fileName, info.Size)
		of, err := os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		if err := download.NewVideoDownloader(info, of).Download(); err != nil {
			log.Infof("download file error: %v", err)
			of.Close()
			os.Remove(fileName)
			panic(err)
		}
		log.Infof("download file %v success", fileName)
		of.Sync()
		of.Close()
	}
	log.Infof("download %v success", id)
}

// 前闭后开区间
type Range struct {
	Start int64
	End   int64
}

func parsePages(pageStr string) (func(int64) bool, error) {
	pageStr = strings.TrimSpace(pageStr)
	if pageStr == "" || pageStr == "*" {
		return func(int64) bool {
			return true
		}, nil
	}

	var rs []*Range
	pages := strings.Split(pageStr, ",")
	for _, page := range pages {
		if strings.Contains(page, "-") {
			// range pages
			pps := strings.Split(page, "-")
			if len(pps) != 2 {
				return nil, fmt.Errorf("not two number: %v", len(pps))
			}
			start, err := strconv.ParseInt(pps[0], 10, 64)
			if err != nil {
				return nil, err
			}
			end, err := strconv.ParseInt(pps[1], 10, 64)
			if err != nil {
				return nil, err
			}
			rs = append(rs, &Range{
				Start: start,
				End:   end,
			})
		} else {
			// single page
			p, err := strconv.ParseInt(page, 10, 64)
			if err != nil {
				return nil, err
			}
			rs = append(rs, &Range{
				Start: p,
				End:   p,
			})
		}
	}

	return func(i int64) bool {
		for _, rg := range rs {
			if i >= rg.Start && i <= rg.End {
				return true
			}
		}
		return false
	}, nil
}
