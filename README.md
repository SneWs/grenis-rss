# grenis-rss
Super basic rss sync tool written in Go

### Usage
This is mostly for my own usage as I like to keep Podcasts etc locally downloaded and not maintained/controlled by some other entity/software such as Apple or Spotify offerings.

The way I use it is simple. I download and sync the podcasts I have an interest in and have Audio bookshelf pick up the downloaded episodes giving me a nice local experience

### Help
```
Usage: grenis-rss
  -h, --help: Show this help message
  -mi, --max-items [integer]: Maximum number of items to download (Set this to -1 if you want to download all files from the feeds configured)
Notes:
  Config file is located at: ~/.config/grenis-rss/config.json
```

### Config file
The config file should be located under ~/.config/grenis-rss/config.json and will define the storage location and what feeds to sync, that's it.
```
{
    "path": "~/Podcasts",
    "feeds": [
        {
            "name": "Late Night Linux",
            "feed_url": "https://latenightlinux.com/feed/mp3"
        }
    ]
}
```

### Libraries
This tool is using the work of:
 * https://github.com/mmcdole/gofeed 