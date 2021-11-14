package download

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/apex/log"
	"github.com/rammiah/bili-downloader/consts"
	"github.com/rammiah/bili-downloader/download/httpcli"
)

// VideoFragment download video in parallel
type VideoFragment struct {
	Begin int64
	End   int64
	Ready Bool
	Data  []byte
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
	out      io.Writer
	frags    []*VideoFragment
	count    int64
	pool     *sync.Pool
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

func NewVideoDownloader(info *DownloadInfo, out io.Writer) *VideoDownloader {
	d := &VideoDownloader{
		downInfo: info,
		wg:       &sync.WaitGroup{},
		out:      out,
		frags:    buildFrags(info),
		downIdx:  0,
		redIdx:   0,
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, consts.FragSize)
			},
		},
		errVal: &atomic.Value{},
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

	buf := bytes.NewBuffer(d.pool.Get().([]byte))
	// buf.Grow(consts.FragSize)
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

		log.Infof("download frag %v success", idx)

		// data for idx is ready
		frag.Data = data
		frag.Ready.Set(true)
		if atomic.LoadInt64(&d.redIdx) == idx {
			log.Infof("start write from %v", idx)
			for idx < int64(len(d.frags)) && d.frags[idx].Ready.Get() {
				d.out.Write(d.frags[idx].Data)
				d.pool.Put(d.frags[idx].Data)
				d.frags[idx].Data = nil
				log.Infof("write %v success", idx)
				idx++
			}
			atomic.StoreInt64(&d.redIdx, idx)
		}
	}
}

func (d *VideoDownloader) Download() error {
	for i := 0; i < runtime.NumCPU(); i++ {
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
