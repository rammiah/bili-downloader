package download

import (
	"fmt"
	"testing"

	"github.com/rammiah/bili-downloader/utils"
	"github.com/stretchr/testify/require"
)

func TestGetVideoInfosById(t *testing.T) {
	const (
		BVId = "BV1pP4y1b7iP"
	)
	urls, err := GetVideoInfosById(BVId)
	require.Nil(t, err)
	fmt.Printf("%v\n", utils.Json(urls))
}
