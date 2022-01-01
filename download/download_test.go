package download

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/download/cookie"
	"github.com/rammiah/bili-downloader/utils"
	"github.com/stretchr/testify/require"
)

const (
	VideoID       = "BV1pP4y1b7iP"
	Avid    int64 = 891245009
	Cid     int64 = 428280666
	MD5           = "2a65e65f2db52d8ddc163c3d7c6cd7b2"
)

func TestMain(m *testing.M) {
	defer cookie.SaveCookies()
	code := m.Run()
	log.Infof("run tests exit code %v", code)
}

func TestGetDownloadInfoByAidCid(t *testing.T) {
	info, err := GetDownloadInfoByAidCid(VideoID, Avid, Cid)
	require.Nil(t, err)
	require.NotNil(t, info)
	require.EqualValues(t, 80, info.Qn)
	require.EqualValues(t, 14382, info.Length)
	require.EqualValues(t, 2955513, info.Size)
	fmt.Printf("download info is: %v\n", utils.Json(info))
}

type WriteCounter int64

func (wc *WriteCounter) Write(data []byte) (int, error) {
	atomic.AddInt64((*int64)(wc), int64(len(data)))
	return len(data), nil
}

func (wc *WriteCounter) GetCount() int64 {
	return atomic.LoadInt64((*int64)(wc))
}

func (wc *WriteCounter) Reset() int64 {
	return atomic.SwapInt64((*int64)(wc), 0)
}

// func TestDownloadVideo(t *testing.T) {
//     info, err := GetDownloadInfoByAidCid(VideoID, Avid, Cid)
//     log.Infof("video %v info %v", VideoID, utils.Json(info))
//     require.Nil(t, err)
//     err = authVideo(VideoID, info.Url)
//     require.Nil(t, err)
//     wc := new(WriteCounter)
//     hash := md5.New()
//     NewVideoDownloader(info, io.MultiWriter(wc, hash)).Download()
//     require.EqualValues(t, info.Size, wc.GetCount())
//     require.Equal(t, MD5, hex.EncodeToString(hash.Sum(nil)))
// }

func TestAuthVideo(t *testing.T) {
	info, err := GetDownloadInfoByAidCid(VideoID, Avid, Cid)
	require.Nil(t, err)
	err = authVideo(VideoID, info.Url)
	require.Nil(t, err)
	if err != nil {
		log.Errorf("auth error: %v", err)
	}
}
