package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Lofter1/anyflip-downloader/anyflip"
	"github.com/br3w0r/goitopdf/itopdf"
)

var title string
var pageCount int
var insecure bool

func init() {
	flag.StringVar(&title, "title", "", "Specifies the name of the generated PDF document (uses book title if not specified)")
	flag.BoolVar(&insecure, "insecure", false, "Skip certificate validation")
}

func main() {
	flag.Parse()
	anyflipURL, err := url.Parse(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if insecure {
		fmt.Println("You enabled insecure downloads. This disables security checks. Stay safe!")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	fmt.Println("Preparing to download")
	err = prepareDownload(anyflipURL)
	if err != nil {
		log.Fatal(err)
	}

	downloadFolder := title
	outputFile := title + ".pdf"

	err = downloadImages(anyflipURL, pageCount, downloadFolder)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Converting to pdf")
	err = createPDF(outputFile, downloadFolder)
	if err != nil {
		log.Fatal(err)
	}

	os.RemoveAll(downloadFolder)
}

func prepareDownload(anyflipURL *url.URL) error {
	sanitizeURL(anyflipURL)
	configjs, err := anyflip.DownloadConfigJSFile(anyflipURL)
	if err != nil {
		return err
	}

	if title == "" {
		title, err = anyflip.GetBookTitle(configjs)
		if err != nil {
			title = path.Base(anyflipURL.String())
		}
	}

	pageCount, err = anyflip.GetPageCount(configjs)
	return err
}

func sanitizeURL(anyflipURL *url.URL) {
	bookURLPathElements := strings.Split(anyflipURL.Path, "/")
	anyflipURL.Path = path.Join("/", bookURLPathElements[1], bookURLPathElements[2])
}

func createPDF(outputFile string, imageDir string) error {
	pdf := itopdf.NewInstance()
	err := pdf.WalkDir(imageDir, nil)
	if err != nil {
		return err
	}
	err = pdf.Save(outputFile)
	if err != nil {
		return err
	}
	return nil
}
