package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rclancey/azlyrics"
)

type argsType struct {
	Artist string
	Name string
}

func (a *argsType) GetArtist() string {
	return a.Artist
}

func (a *argsType) GetName() string {
	return a.Name
}

func main() {
	args := &argsType{}
	flag.StringVar(&args.Artist, "artist", "", "artist name")
	flag.StringVar(&args.Name, "name", "", "song name")
	flag.Parse()

	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "azlyrics")
	cacheTime := time.Hour * 24 * 30
	c, err := azlyrics.NewLyricsClient(cacheDir, cacheTime)
	if err != nil {
		log.Fatal(err)
	}
	search, err := c.Search(args)
	if err != nil {
		log.Fatal(err)
	}
	if len(search.Results) == 1 {
		err = c.LoadResult(search.Results[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*search.Results[0].Lyrics)
	} else {
		for i, m := range search.Results {
			fmt.Printf("%d: %s / %s\n", i + 1, m.Artist, m.Song)
		}
		fmt.Printf("Pick one: ")
		var idx int
		_, err = fmt.Scanln(&idx)
		if err != nil {
			log.Fatal(err)
		}
		m := search.Results[idx - 1]
		err = c.LoadResult(m)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*m.Lyrics)
	}
}
