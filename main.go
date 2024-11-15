package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	feed "github.com/mmcdole/gofeed"
)

type FeedConfig struct {
	FeedItems []FeedItem `json:"feeds"`
	Path      string     `json:"path"`
}

type FeedItem struct {
	Name    string `json:"name"`
	FeedURL string `json:"feed_url"`
}

func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configFolder := homeDir + "/.config/grenis-rss"
	_ = os.MkdirAll(configFolder, os.FileMode.Perm(0755))

	return configFolder
}

func getConfigFile() string {
	configFolder := getConfigDir()
	return configFolder + "/config.json"
}

func readFeedsConfig() (*FeedConfig, error) {
	configFilename := getConfigFile()
	configFile, err := os.Open(configFilename)
	if err != nil {
		return nil, err
	}

	defer func(configFile *os.File) {
		_ = configFile.Close()
	}(configFile)

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		panic(err)
	}

	var feedConfig = FeedConfig{}
	err = json.Unmarshal(configBytes, &feedConfig)
	if err != nil {
		return nil, err
	}

	return &feedConfig, nil
}

func createDefaultConfig() {
	configFilename := getConfigFile()

	// Check if file exists, and if it does, don't overwrite it
	if _, err := os.Stat(configFilename); err == nil {
		return
	}

	configFile, err := os.Create(configFilename)
	if err != nil {
		panic(err)
	}
	defer func(configFile *os.File) {
		_ = configFile.Close()
	}(configFile)

	var feedConfig = FeedConfig{
		FeedItems: []FeedItem{},
		Path:      "~/Podcasts",
	}

	configBytes, err := json.Marshal(feedConfig)
	if err != nil {
		panic(err)
	}

	_, _ = configFile.Write(configBytes)
}

func makeAbsolute(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	if path == "~" {
		path = homeDir
	}
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}

	return path
}

func basicSanitizeFilename(filename string) string {
	str := strings.Replace(filename, "/", "_", -1)
	str = strings.Replace(str, ":", "", -1)
	str = strings.Replace(str, "$", "", -1)
	str = strings.Replace(str, "#", "", -1)
	str = strings.Replace(str, "€", "", -1)
	str = strings.Replace(str, "£", "", -1)
	str = strings.Replace(str, "!", "", -1)
	str = strings.Replace(str, "?", "", -1)
	str = strings.Replace(str, "+", "", -1)
	str = strings.Replace(str, "&", "", -1)
	str = strings.Replace(str, "*", "", -1)
	str = strings.Replace(str, "@", "", -1)
	str = strings.Replace(str, "(", "", -1)
	str = strings.Replace(str, ")", "", -1)
	str = strings.Replace(str, "`", "", -1)
	str = strings.Replace(str, "'", "", -1)
	str = strings.Replace(str, ",", "", -1)
	str = strings.Replace(str, "\"", "", -1)

	return str
}

func processFeedUrl(savePath string, feedUrl string, maxItems int) {
	fp := feed.NewParser()
	parsedUrl, err := fp.ParseURL(feedUrl)

	if err != nil {
		fmt.Println("Failed to parse parsedUrl:", feedUrl, "Error:", err)
	}

	if parsedUrl == nil {
		fmt.Println("Failed to parse parsedUrl URL:", feedUrl)
		return
	}

	var count = 0
	for _, item := range parsedUrl.Items {

		if maxItems > 0 && count >= maxItems {
			break
		}

		count += 1

		if len(item.Enclosures) < 1 {
			continue
		}

		enc := item.Enclosures[0]
		r, _ := http.NewRequest("GET", enc.URL, nil)
		ext := filepath.Ext(path.Base(r.URL.Path))

		localFilename := basicSanitizeFilename(item.Title) + ext
		filename := savePath + "/" + localFilename
		file, err := os.Open(filename)
		if err == nil {
			// File already exists, skip
			err := file.Close()
			if err != nil {
				_ = fmt.Errorf("Failed to close file: %v", err)
			}
			continue
		}

		file, err = os.Create(filename)
		if err != nil {
			fmt.Println("Failed to create file for:", filename, "Error:", err)
			continue
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				_ = fmt.Errorf("Failed to close file: %v", err)
			}
		}(file)

		resp, err := http.Get(enc.URL)
		if err != nil {
			fmt.Println("Failed to download file:", enc.URL, "Error:", err)
			continue
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				_ = fmt.Errorf("Failed to close file: %v", err)
			}
		}(resp.Body)
		fmt.Println("Downloading", enc.URL)

		bytes, err := io.Copy(file, resp.Body)
		if err != nil {
			fmt.Println("Failed to save file:", filename, "Error:", err)
			continue
		}

		fmt.Println("Download completed for", filename, "Size:", bytes)
	}
}

func printHelp() {
	fmt.Println("Usage: grenis-rss")
	fmt.Println("  -h, --help: Show this help message")
	fmt.Println("  -mi, --max-items [integer]: Maximum number of items to download")
	fmt.Println("Notes:")
	fmt.Println("  Config file is located at: ~/.config/grenis-rss/config.json")
}

func main() {
	createDefaultConfig()
	maxItems := 1 // Default to only taking the most recent episode

	args := os.Args
	for i, arg := range args {
		if arg == "-h" || arg == "--help" {
			printHelp()
			return
		}
		if arg == "-mi" || arg == "--max-items" {
			ic, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("Invalid max items value:", args[i+1])
				printHelp()
				return
			}

			maxItems = ic
		}
	}

	feedConfig, err := readFeedsConfig()
	if err != nil {
		fmt.Println("Failed to read config file:", err)
		printHelp()
		return
	}

	feedPath := makeAbsolute(feedConfig.Path)

	// Make sure our output root exists
	err = os.MkdirAll(feedPath, os.FileMode.Perm(0755))
	if err != nil {
		panic(err)
	}

	if len(feedConfig.FeedItems) < 1 {
		fmt.Println("No feeds configured")
		printHelp()
		return
	}

	var wg sync.WaitGroup

	for _, feedItem := range feedConfig.FeedItems {
		wg.Add(1)

		fmt.Println("Starting to process feed:", feedItem.Name)
		folderName := feedPath + "/" + feedItem.Name
		err := os.MkdirAll(folderName, os.FileMode.Perm(0755))
		if err != nil {
			panic(err)
			return
		}

		go func(url string, items int) {
			defer wg.Done()
			processFeedUrl(folderName, url, items)
		}(feedItem.FeedURL, maxItems)
	}

	wg.Wait()
	fmt.Println("Done")
}
