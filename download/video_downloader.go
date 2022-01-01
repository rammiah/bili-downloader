package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/consts"
	"github.com/rammiah/bili-downloader/download/httpcli"
)

// VideoFragment download video in parallel
type VideoFragment struct {
	Begin int64
	End   int64
}

type Bool int32

func (b *Bool) Get() bool {
	return atomic.LoadInt32((*int32)(b)) == 1
}

func (b *Bool) Set(val bool) {
	if val {
		atomic.StoreInt32((*int32)(b), 1)
		return
	}
	atomic.StoreInt32((*int32)(b), 0)
}

func (b *Bool) Clear() {
	atomic.StoreInt32((*int32)(b), 0)
}

type VideoDownloader struct {
	downInfo *DownloadInfo
	wg       *sync.WaitGroup
	out      *os.File
	frags    []*VideoFragment
	count    int64
	errVal   *atomic.Value

	downIdx int64
	redIdx  int64
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

func NewVideoDownloader(info *DownloadInfo, out *os.File) *VideoDownloader {
	syscall.Fallocate(int(out.Fd()), 0, 0, info.Size)
	d := &VideoDownloader{
		downInfo: info,
		wg:       &sync.WaitGroup{},
		out:      out,
		frags:    buildFrags(info),
		downIdx:  0,
		errVal:   &atomic.Value{},
	}
	d.count = int64(len(d.frags))

	return d
}

func (d *VideoDownloader) DownloadFragment(frag *VideoFragment) ([]byte, error) {
	info := d.downInfo

	// auth audio
	if err := authVideo(info.VideoID, info.Url); err != nil {
		log.Errorf("auth video error: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.Url, nil)
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

	buf := bytes.NewBuffer(nil)
	buf.Grow(consts.FragSize)
	buf.Reset()

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.Errorf("read resp data error: %v", err)
		return nil, err
	}

	// log.Infof("download size %v bytes", buf.Len())

	return buf.Bytes(), nil
}

func (d *VideoDownloader) startWorker(id int) {
	defer d.wg.Done()
	for {
		// check error
		if err := d.errVal.Load(); err != nil {
			log.Infof("error detected: %v", err)
			return
		}
		idx := atomic.AddInt64(&d.downIdx, 1) - 1
		if idx >= int64(len(d.frags)) {
			log.Infof("worker %v exit", id)
			return
		}

		frag := d.frags[idx]
		// random sleep
		time.Sleep(time.Duration(rand.Intn(100)+200) * time.Millisecond)
		log.Infof("download frag %v, %v - %v", idx, frag.Begin, frag.End)
		data, err := d.DownloadFragment(frag)
		if err != nil {
			log.Errorf("download %v error: %v", idx, err)
			d.errVal.Store(err)
			return
		}

		if err := d.errVal.Load(); err != nil {
			log.Infof("error detected: %v\n", err)
			return
		}

		_, err = d.out.WriteAt(data, frag.Begin)
		if err != nil {
			log.Errorf("write file error: %v", err)
			d.errVal.Store(err)
			return
		}

		log.Infof("download frag %v success", idx)
	}
}

func (d *VideoDownloader) Download() error {
	DownThreadCnt := runtime.NumCPU()
	for i := 0; i < DownThreadCnt; i++ {
		d.wg.Add(1)
		go d.startWorker(i)
	}
	d.wg.Wait()
	if err := d.errVal.Load(); err != nil {
		log.Infof("download failed, error: %v", err)
		return err.(error)
	}
	log.Infof("download success")
	return nil
}
