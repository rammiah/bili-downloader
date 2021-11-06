package download

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/consts"
	"github.com/rammiah/bili-downloader/download/httpcli"
)

// VideoFragment download video in parallel
type VideoFragment struct {
	Begin int64
	End   int64
	Ready bool
	Data  []byte
}

type VideoDownloader struct {
	downInfo *DownloadInfo
	wg       *sync.WaitGroup
	out      io.Writer
	frags    []*VideoFragment
	lock     *sync.Mutex
	// cond     *sync.Cond

	downIdx    int
	unreadyIdx int
}

func buildFrags(info *DownloadInfo) []*VideoFragment {
	fragCnt := info.Size/consts.FragSize + 1
	if info.Size%consts.FragSize == 0 {
		fragCnt--
	}

	frags := make([]*VideoFragment, 0, fragCnt)
	for i := 0; i < int(fragCnt); i++ {
		frag := &VideoFragment{
			Begin: int64(i * consts.FragSize),
			End:   int64((i+1)*consts.FragSize) - 1,
		}
		if i == int(fragCnt-1) {
			frag.End = info.Size - 1
		}
		frags = append(frags, frag)
	}
	log.Infof("frags count %v", len(frags))
	return frags
}

func NewVideoDownloader(info *DownloadInfo, out io.Writer) *VideoDownloader {
	lk := &sync.Mutex{}
	return &VideoDownloader{
		downInfo: info,
		wg:       &sync.WaitGroup{},
		out:      out,
		frags:    buildFrags(info),
		lock:     lk,
		// cond: &sync.Cond{
		//     L: lk,
		// },
		downIdx:    0,
		unreadyIdx: 0,
	}
}

func (d *VideoDownloader) DownloadFragment(frag *VideoFragment) ([]byte, error) {
	info := d.downInfo

	// auth audio
	if err := authVideo(info.VideoID, info.Url); err != nil {
		log.Errorf("auth video error: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, info.Url, nil)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"accept":             "*/*",
		"accept-encoding":    "identity",
		"accept-language":    "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
		"dnt":                "1",
		"origin":             "https://www.bilibili.com",
		"referer":            "https://www.bilibili.com/video/" + info.VideoID,
		"sec-ch-ua":          `"Google Chrome";v="95", "Chromium";v="95", ";Not A Brand";v="99"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "Windows",
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "cross-site",
		"user-agent":         httpcli.UA,
	}

	for k, v := range params {
		req.Header.Set(k, v)
	}

	// set range header
	req.Header.Set("range", fmt.Sprintf("bytes=%v-%v", frag.Begin, frag.End))

	resp, err := httpcli.Inst.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("get status not ok: %v", resp.Status)
	}
	if resp.ContentLength != frag.End-frag.Begin+1 {
		return nil, fmt.Errorf("content size not same: expect %v got %v", info.Size, resp.ContentLength)
	}
	// log.Infof("download request success")

	buf := bytes.NewBuffer(nil)
	buf.Grow(consts.FragSize)

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.Errorf("read resp data error: %v", err)
		return nil, err
	}

	// log.Infof("download size %v bytes", buf.Len())

	return buf.Bytes(), nil

}

func (d *VideoDownloader) startWorker() {
	defer d.wg.Done()
	for {
		d.lock.Lock()
		if d.downIdx == len(d.frags) {
			// worker download mission done
			// log.Infof("worker download done")
			d.lock.Unlock()
			return
		}
		idx := d.downIdx
		d.downIdx++
		d.lock.Unlock()
		frag := d.frags[idx]
		log.Infof("download frag %v, %v - %v", idx, frag.Begin, frag.End)
		data, err := d.DownloadFragment(frag)
		if err != nil {
			log.Errorf("download error: %v", err)
			return
		}
		log.Infof("download frag %v success", idx)
		// 看下这个是不是可以写到target
		d.lock.Lock()
		frag.Ready = true
		frag.Data = data
		if d.unreadyIdx == idx {
			log.Infof("start write from %v", idx)
			// 是的，该我写了，并且要更新readyIdx
			for d.unreadyIdx < d.downIdx && d.frags[d.unreadyIdx].Ready {
				d.out.Write(d.frags[d.unreadyIdx].Data)
				d.frags[d.unreadyIdx].Data = nil // for gc
				log.Infof("write %v success", d.unreadyIdx)
				d.unreadyIdx++
			}
		}
		d.lock.Unlock()
	}
}

func (d *VideoDownloader) Download() {
	for i := 0; i < runtime.NumCPU(); i++ {
		d.wg.Add(1)
		go d.startWorker()
	}
	d.wg.Wait()
	log.Infof("download success")
}
