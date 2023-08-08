package anyflip

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/schollz/progressbar/v3"
)

func DownloadImages(url *url.URL, pageCount int, downloadFolder string) error {
	err := os.Mkdir(downloadFolder, os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(pageCount,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Downloading"),
	)
	downloadURL, err := url.Parse("https://online.anyflip.com")
	if err != nil {
		return err
	}

	for page := 1; page <= pageCount; page++ {
		downloadURL.Path = path.Join(url.Path, "files", "mobile", strconv.Itoa(page)+".jpg")
		response, err := http.Get(downloadURL.String())
		if err != nil {
			return err
		}

		if response.StatusCode != http.StatusOK {
			return errors.New("Received non-200 response: " + response.Status)
		}

		extension := path.Ext(downloadURL.String())
		filename := fmt.Sprintf("%04d%v", page, extension)
		file, err := os.Create(path.Join(downloadFolder, filename))
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return err
		}

		bar.Add(1)
	}
	fmt.Println()
	return nil
}
