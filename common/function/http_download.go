package function

import (
	"context"

	"github.com/cavaliergopher/grab/v3"
)

const downloadUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"

func DownloadFile(ctx context.Context, url string, saveFilePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	client := grab.NewClient()
	client.UserAgent = downloadUserAgent

	req, err := grab.NewRequest(saveFilePath, url)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp := client.Do(req)
	return resp.Err()
}
