package function

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/shirou/gopsutil/disk"
)

func DownloadFile(ctx context.Context, url string, saveFilePath string, progress func(float64)) error {
	if progress == nil {
		progress = func(_ float64) {}
	}
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	contentLength, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		contentLength = 0
	}

	client := grab.NewClient()
	req, _ := grab.NewRequest(saveFilePath, url)
	req = req.WithContext(ctx)
	req.HTTPRequest.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Size = int64(contentLength)
	req.CanResume = true

	// start download
	fmt.Printf("Downloading to %v...\n", saveFilePath)

	resp := client.Do(req)

	// start download
	fmt.Printf("size %v...\n", resp.Size())

	needDiskSize := req.Size
	fi, err := os.Stat(saveFilePath)
	if err == nil {
		needDiskSize = needDiskSize - fi.Size()
	}
	needDiskSize = needDiskSize + req.Size/2
	diskDir := saveFilePath
	usage, err := disk.Usage(diskDir)
	slog.Info("disk usage", "path", diskDir, "free", usage, "err", err)
	if err == nil && usage.Free < uint64(needDiskSize) {
		return errors.New("磁盘空间不足, 请更改安装目录")
	}

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			percentage := resp.Progress() * 100
			progress(percentage)

		case <-resp.Done:
			// download is complete
			break Loop
		}
	}

	return resp.Err()
}
