package download

import (
	"fmt"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/download/cookie"
	"github.com/rammiah/bili-downloader/utils"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	defer cookie.SaveCookies()
	code := m.Run()
	log.Infof("run tests exit code %v", code)
}

func TestGetDownloadInfoByAidCid(t *testing.T) {
	var (
		avid int64 = 891245009
		cid  int64 = 428280666
	)
	info, err := GetDownloadInfoByAidCid(avid, cid)
	require.Nil(t, err)
	require.NotNil(t, info)
	require.EqualValues(t, 80, info.Qn)
	require.EqualValues(t, 14382, info.Length)
	require.EqualValues(t, 2955513, info.Size)
	fmt.Printf("download info is: %v\n", utils.Json(info))
}

func TestDownloadVideo(t *testing.T) {
	const (
		URL = "https://upos-sz-mirrorcoso1.bilivideo.com/upgcxcode/63/12/428591263/428591263-1-80.flv?e=ig8euxZM2rNcNbNzhwdVhwdlhbhVhwdVhoNvNC8BqJIzNbfqXBvEqxTEto8BTrNvN0GvT90W5JZMkX_YN0MvXg8gNEV4NC8xNEV4N03eN0B5tZlqNxTEto8BTrNvNeZVuJ10Kj_g2UB02J0mN0B5tZlqNCNEto8BTrNvNC7MTX502C8f2jmMQJ6mqF2fka1mqx6gqj0eN0B599M=&uipk=5&nbs=1&deadline=1636051791&gen=playurlv2&os=coso1bv&oi=2070954129&trid=8374c8d0169547efb5f7ba624eb1e861u&platform=pc&upsig=b5499fe13bcccf748583128411e3a444&uparams=e,uipk,nbs,deadline,gen,os,oi,trid,platform&mid=101994113&bvc=vod&nettype=0&orderid=0,3&agrr=1&logo=80000000"
	)

	err := DownloadVideo(&DownloadInfo{
		Url: URL,
	}, io.Discard)
	require.Nil(t, err)
}
