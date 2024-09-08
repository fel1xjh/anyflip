package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfpkg/pdfcpu"
	"github.com/schollz/progressbar/v3"
)

var title string
var insecure bool
var keepDownloadFolder bool

type flipbook struct {
	URL       *url.URL
	title     string
	pageCount int
	pageURLs  []string
}

func init() {
	flag.Usage = printUsage
	flag.StringVar(&title, "title", "", "Specifies the name of the generated PDF document (uses book title if not specified)")
	flag.BoolVar(&insecure, "insecure", false, "Skip certificate validation")
	flag.BoolVar(&keepDownloadFolder, "keep-download-folder", false, "Keep the temporary download folder instead of deleting it after completion")
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
	flipbook, err := prepareDownload(anyflipURL)
	if err != nil {
		log.Fatal(err)
	}

	outputFile := title + ".pdf"

	err = flipbook.downloadImages("D:/Downloads")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Converting to pdf")
	err = createPDF(outputFile, "D:/Downloads")
	if err != nil {
		log.Fatal(err)
	}

	if !keepDownloadFolder {
		os.RemoveAll("D:/Downloads")
	}
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s [OPTIONS] <url>\n", os.Args[0])
	fmt.Fprintf(w, "Options:\n")
	flag.PrintDefaults()
}

func prepareDownload(anyflipURL *url.URL) (*flipbook, error) {
	var newFlipbook flipbook

	sanitizeURL(anyflipURL)
	newFlipbook.URL = anyflipURL

	configjs, err := downloadConfigJSFile(anyflipURL)
	if err != nil {
		return nil, err
	}

	if title == "" {
		title, err = getBookTitle(configjs)
		if err != nil {
			title = path.Base(anyflipURL.String())
		}
	}

	newFlipbook.title = govalidator.SafeFileName(title)
	newFlipbook.pageCount, err = getPageCount(configjs)
	pageFileNames := getPageFileNames(configjs)

	downloadURL, _ := url.Parse("https://online.anyflip.com/")
	println(newFlipbook.URL.String())
	if len(pageFileNames) == 0 {
		for i := 1; i <= newFlipbook.pageCount; i++ {
			downloadURL.Path = path.Join(newFlipbook.URL.Path, "files", "mobile", strconv.Itoa(i)+".jpg")
			newFlipbook.pageURLs = append(newFlipbook.pageURLs, downloadURL.String())
		}
	} else {
		for i := 0; i < newFlipbook.pageCount; i++ {
			downloadURL.Path = path.Join(newFlipbook.URL.Path, "files", "large", pageFileNames[i])
			newFlipbook.pageURLs = append(newFlipbook.pageURLs, downloadURL.String())
		}
	}

	return &newFlipbook, err
}

func sanitizeURL(anyflipURL *url.URL) {
	bookURLPathElements := strings.Split(anyflipURL.Path, "/")
	anyflipURL.Path = path.Join("/", bookURLPathElements[1], bookURLPathElements[2])
}

func createPDF(outputFile string, imageDir string) error {
	outputFile = strings.ReplaceAll(outputFile, "'", "")
	outputFile = strings.ReplaceAll(outputFile, "\\", "")
	outputFile = strings.ReplaceAll(outputFile, ":", "")

	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Output file %s already exists", outputFile)
		return nil
	}

	files, err := os.ReadDir(imageDir)
	if err != nil {
		return err
	}

	var imagePaths []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			imagePaths = append(imagePaths, filepath.Join(imageDir, file.Name()))
		}	}

	api.NewContext("pdfcpu")
	pdf, err := api.ImportFldr(imagePaths)
	if err != nil {
		return err
	}

	err = pdf.WriteFile(outputFile)
	if err != nil {
		return err
	}

	fmt.Printf("PDF created: %s\n", outputFile)
	return nil
}

func downloadConfigJSFile(anyflipURL *url.URL) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", anyflipURL.String()+"?configjs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func getBookTitle(configjs []byte) (string, error) {
	var config map[string]interface{}
	err := json.Unmarshal(configjs, &config)
	if err != nil {
		return "", err
	}

	title, ok := config["bookTitle"]
	if !ok {
		return "", errors.New("book title not found")
	}

	return title.(string), nil
}

func getPageCount(configjs []byte) (int, error) {
	var config map[string]interface{}
	err := json.Unmarshal(configjs, &config)
	if err != nil {
		return 0, err
	}

	pageCount, ok := config["pageCount"]
	if !ok {
		return 0, errors.New("page count not found")
	}

	return int(pageCount.(float64)), nil
}

func getPageFileNames(configjs []byte) []string {
	var config map[string]interface{}
	err := json.Unmarshal(configjs, &config)
	if err != nil {
		return nil
	}

	pageFileNames, ok := config["pageFileNames"]
	if !ok {
		return nil
	}

	pageFileNamesSlice, ok := pageFileNames.([]interface{})
	if !ok {
		return nil
	}

	var pageFileNamesStr []string
	for _, v := range pageFileNamesSlice {
		pageFileNamesStr = append(pageFileNamesStr, v.(string))
	}

	return pageFileNamesStr
}

func (f *flipbook) downloadImages(downloadDir string) error {
	bar := progressbar.Default(int64(f.pageCount))
	for _, url := range f.pageURLs {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		fileName := filepath.Base(url)
		filePath := filepath.Join(downloadDir, fileName)
		f, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return err
		}

		bar.Add(1)
	}

	return nil
}
