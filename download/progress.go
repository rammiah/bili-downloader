package download

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rammiah/bili-downloader/consts"
)

type ProgressBar struct {
	Total       int64
	Done        int64
	lastPercent int64
	stopOnce    sync.Once
	stop        chan struct{}
}

func NewProgressBar(total int64, wg *sync.WaitGroup) *ProgressBar {
	p := &ProgressBar{
		Total: total,
		stop:  make(chan struct{}, 1),
	}

	wg.Add(1)
	go p.print(wg)

	return p
}

func (p *ProgressBar) print(wg *sync.WaitGroup) {
	defer func() {
		// fmt.Printf("progress bar exited\n")
		wg.Done()
	}()
	ticker := time.NewTicker(time.Millisecond * 30)
	for p.lastPercent < 100 {
		select {
		case <-ticker.C:
		case <-p.stop:
			os.Stdout.WriteString("\n")
			return
		}
		done := atomic.LoadInt64(&p.Done)
		percent := int(100.0 * float64(done) / float64(p.Total))
		if done >= p.Total {
			percent = 100
		}
		// if percent > int(p.lastPercent) {
		cmd := strings.Repeat("\b", 200) + consts.Byte(done).String() + "/" + consts.Byte(p.Total).String() +
			" [" + strings.Repeat("#", percent) + strings.Repeat(" ", 100-percent) + "]"
		if percent >= 100 {
			cmd += "\n"
		}
		os.Stdout.WriteString(cmd)
		os.Stdout.Sync()
		p.lastPercent = int64(percent)
		// }
	}
}

func (p *ProgressBar) Stop() {
	p.stopOnce.Do(func() {
		close(p.stop)
	})
}

func (p *ProgressBar) Add(val int64) {
	val = atomic.AddInt64(&p.Done, val)
}

func (p *ProgressBar) Write(buf []byte) (int, error) {
	p.Add(int64(len(buf)))

	return len(buf), nil
}
