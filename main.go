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

func get_config_dir() string {
	home_dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	config_folder := home_dir + "/.config/grenis-rss"
	os.MkdirAll(config_folder, os.FileMode.Perm(0755))

	return config_folder
}

func get_config_file() string {
	config_folder := get_config_dir()
	return config_folder + "/config.json"
}

func read_feeds_config() (*FeedConfig, error) {
	config_filename := get_config_file()
	config_file, err := os.Open(config_filename)
	if err != nil {
		return nil, err
	}
	defer config_file.Close()

	config_bytes, err := io.ReadAll(config_file)
	if err != nil {
		panic(err)
	}

	var feed_config = FeedConfig{}
	json.Unmarshal(config_bytes, &feed_config)

	return &feed_config, nil
}

func create_default_config() {
	config_filename := get_config_file()

	// Check if file exists, and if it does, don't overwrite it
	if _, err := os.Stat(config_filename); err == nil {
		return
	}

	config_file, err := os.Create(config_filename)
	if err != nil {
		panic(err)
	}
	defer config_file.Close()

	var feed_config = FeedConfig{
		FeedItems: []FeedItem{},
		Path:      "~/Podcasts",
	}

	config_bytes, err := json.Marshal(feed_config)
	if err != nil {
		panic(err)
	}

	config_file.Write(config_bytes)
}

func makeAbsolute(path string) string {
	home_dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	if path == "~" {
		path = home_dir
	}
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home_dir, path[2:])
	}

	return path
}

func basic_sanitize_filename(filename string) string {
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
	str = strings.Replace(str, "'", "", -1)
	str = strings.Replace(str, ",", "", -1)
	str = strings.Replace(str, "\"", "", -1)

	return str
}

func process_feed_url(save_path string, feed_url string, max_items int) {
	fp := feed.NewParser()
	feed, err := fp.ParseURL(feed_url)

	if err != nil {
		fmt.Println("Failed to parse feed:", feed_url, "Error:", err)
	}

	if feed == nil {
		fmt.Println("Failed to parse feed URL:", feed_url)
		return
	}

	var count int = 0
	for _, item := range feed.Items {

		if max_items > 0 && count >= max_items {
			break
		}

		count += 1

		if len(item.Enclosures) < 1 {
			continue
		}

		enc := item.Enclosures[0]
		r, _ := http.NewRequest("GET", enc.URL, nil)
		ext := filepath.Ext(path.Base(r.URL.Path))

		local_filename := basic_sanitize_filename(item.Title) + ext
		filename := save_path + "/" + local_filename
		file, err := os.Open(filename)
		if err == nil {
			// File already exists, skip
			file.Close()
			continue
		}

		file, err = os.Create(filename)
		if err != nil {
			fmt.Println("Failed to create file for:", filename, "Error:", err)
			continue
		}
		defer file.Close()

		resp, err := http.Get(enc.URL)
		if err != nil {
			fmt.Println("Failed to download file:", enc.URL, "Error:", err)
			continue
		}

		defer resp.Body.Close()
		fmt.Println("Downloading", enc.URL)

		bytes, err := io.Copy(file, resp.Body)
		if err != nil {
			fmt.Println("Failed to save file:", filename, "Error:", err)
			continue
		}

		fmt.Println("Download completed for", filename, "Size:", bytes)
	}
}

func print_help() {
	fmt.Println("Usage: grenis-rss")
	fmt.Println("  -h, --help: Show this help message")
	fmt.Println("  -mi, --max-items [integer]: Maximum number of items to download")
	fmt.Println("Notes:")
	fmt.Println("  Config file is located at: ~/.config/grenis-rss/config.json")
}

func main() {
	create_default_config()
	max_items := 1 // Default to only taking the most recent episode

	args := os.Args
	for i, arg := range args {
		if arg == "-h" || arg == "--help" {
			print_help()
			return
		}
		if arg == "-mi" || arg == "--max-items" {
			ic, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("Invalid max items value:", args[i+1])
				print_help()
				return
			}

			max_items = ic
		}
	}

	feed_config, err := read_feeds_config()
	if err != nil {
		fmt.Println("Failed to read config file:", err)
		print_help()
		return
	}

	feedPath := makeAbsolute(feed_config.Path)

	// Make sure our output root exists
	err = os.MkdirAll(feedPath, os.FileMode.Perm(0755))
	if err != nil {
		panic(err)
	}

	if len(feed_config.FeedItems) < 1 {
		fmt.Println("No feeds configured")
		print_help()
		return
	}

	var wg sync.WaitGroup

	for _, feed_item := range feed_config.FeedItems {
		wg.Add(1)

		fmt.Println("Starting to process feed:", feed_item.Name)
		folder_name := feedPath + "/" + feed_item.Name
		os.MkdirAll(folder_name, os.FileMode.Perm(0755))

		go func(url string, items int) {
			defer wg.Done()
			process_feed_url(folder_name, url, items)
		}(feed_item.FeedURL, max_items)
	}

	wg.Wait()
	fmt.Println("Done")
}
