package main

import (
	"flag"
	"fmt"

	"github.com/rammiah/bili-downloader/download"
	"github.com/rammiah/bili-downloader/download/cookie"
	"github.com/rammiah/bili-downloader/utils"
)

func main() {
	defer cookie.SaveCookies()
	var (
		id string
	)
	flag.StringVar(&id, "id", "BV1YL4y1q7yi", "video id like avxxx/BVxxx")
	flag.Parse()
	resp, err := download.GetVideoInfosById(id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", utils.Json(resp))
	for _, video := range resp {
		info, err := download.GetDownloadInfoByAidCid(id, video.Avid, video.Cid)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v\n", utils.Json(info))
	}
}
