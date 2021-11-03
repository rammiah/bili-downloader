package playurl

import (
	"fmt"
	"testing"

	"github.com/rammiah/bili-downloader/utils"
	"github.com/stretchr/testify/require"
)

func TestGetPlayUrlsById(t *testing.T) {
	const (
		BVId = "BV16b4y1b7fH"
	)
	urls, err := GetPlayUrlsById(BVId)
	require.Nil(t, err)
	fmt.Printf("%v\n", utils.Json(urls))
}
