package main

import (
	"fmt"

	"github.com/rammiah/bili-downloader/playurl"
	"github.com/rammiah/bili-downloader/utils"
)

func main() {
	resp, err := playurl.GetPlayUrlsById("BV123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", utils.Json(resp))
}
