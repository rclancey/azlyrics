package azlyrics

import (
	"bytes"
	"fmt"
	"io/ioutil"
	//"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	//"github.com/pkg/errors"

	"github.com/rclancey/apiclient"
	"github.com/rclancey/cache/fs"
	"github.com/rclancey/itunes/persistentId"
)

type Track interface {
	GetArtist() string
	GetName() string
}

type LyricsClient struct {
	client *apiclient.APIClient
}

func NewLyricsClient(cacheDir string, cacheTime time.Duration) (*LyricsClient, error) {
	opts := apiclient.APIClientOptions{
		BaseURL: "https://www.azlyrics.com/",
		RequestTimeout: 0,
		CacheStore: fscache.NewFSCacheStore(cacheDir),
		MaxCacheTime: cacheTime,
		MaxRequestsPerSecond: 4.0,
	}
	api, err := apiclient.NewAPIClient(opts)
	if err != nil {
		return nil, err
	}
	client := &LyricsClient{
		client: api,
	}
	return client, nil
}

type Lyrics struct {
	PersistentID pid.PersistentID `json:"persistent_id,omitempty" db:"id"`
	Search       string           `json:"search" db:"search"`
	Lyrics       *string          `json:"lyrics" db:"lyrics"`
}

type LyricsSearchResult struct {
	Artist string  `json:"artist"`
	Song   string  `json:"song"`
	Search string  `json:"search"`
	URL    string  `json:"url"`
	Lyrics *string `json:"lyrics"`
}

type LyricsSearch struct {
	Artist  string `json:"artist"`
	Song    string `json:"song"`
	Search  string `json:"search"`
	Results []*LyricsSearchResult `json:"results"`
}

func (c *LyricsClient) Search(t Track) (*LyricsSearch, error) {
	search := &LyricsSearch{
		Artist: t.GetArtist(),
		Song:   t.GetName(),
	}
	search.Search = fmt.Sprintf("%s %s", strings.ToLower(search.Artist), strings.ToLower(search.Song))
	q := url.Values{"q": []string{search.Search}}
	u := &url.URL{
		Scheme: "https",
		Host: "search.azlyrics.com",
		Path: "/search.php",
		RawQuery: q.Encode(),
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	doc.Find("table.table.table-condensed tr td a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		parts := []string{}
		s.Find("b").Each(func(j int, b *goquery.Selection) {
			parts = append(parts, b.Text())
		})
		if len(parts) >= 2 {
			res := &LyricsSearchResult{
				Artist: parts[1],
				Song:   parts[0],
				Search: search.Search,
				URL:    href,
			}
			search.Results = append(search.Results, res)
		}
	})
	/*
	n := 3
	if len(search.Results) < n {
		n = len(search.Results)
	}
	for _, res := range search.Results[:n] {
		c.LoadResult(res)
	}
	*/
	return search, nil
}

func (c *LyricsClient) LoadResult(r *LyricsSearchResult) error {
	if r.Lyrics != nil {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, r.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/357.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	found := false
	nodes := doc.Find("div.main-page div.row div.text-center div.lyricsh")
	nodes.Parent().ChildrenFiltered("div").Each(func(i int, s *goquery.Selection) {
		if found {
			return
		}
		cls, _ := s.Attr("class")
		if cls == "" {
			t := strings.TrimSpace(s.Text())
			r.Lyrics = &t
			found = true
		}
	})
	if !found {
		return fmt.Errorf("lyrics not found in %s", r.URL)
	}
	return nil
}
