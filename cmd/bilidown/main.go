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
		log.Infof("process avid %v, cid %v", video.Avid, video.Cid)
		info, err := download.GetDownloadInfoByAidCid(id, video.Avid, video.Cid)
		if err != nil {
			panic(err)
		}
		fileName := video.Part + "." + info.Format
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
