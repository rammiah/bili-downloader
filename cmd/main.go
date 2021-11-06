package main

import (
	"flag"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/download"
	"github.com/rammiah/bili-downloader/download/cookie"
)

func main() {
	defer cookie.SaveCookies()
	var (
		id string
	)
	flag.StringVar(&id, "id", "", "video id like avxxx/BVxxx")
	flag.Parse()
	id = strings.TrimSpace(id)
	if id == "" {
		flag.Usage()
		os.Exit(-1)
	}
	resp, err := download.GetVideoInfosById(id)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%v\n", utils.Json(resp))
	for _, video := range resp {
		info, err := download.GetDownloadInfoByAidCid(id, video.Avid, video.Cid)
		if err != nil {
			panic(err)
		}
		log.Infof("start download file %v.%v, size %v bytes", video.Part, info.Format, info.Size)
		of, err := os.OpenFile(video.Part+"."+info.Format, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		download.NewVideoDownloader(info, of).Download()
		// ; err != nil {
		//     of.Close()
		//     panic(err)
		// }
		log.Infof("download file %v.%v success", info.Format, video.Part)
		of.Sync()
		of.Close()
	}
	log.Infof("download %v success", id)
}
