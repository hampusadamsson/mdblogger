package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/feeds"
)

type Rss struct {
	BlogName        string
	BlogURL         string
	AuthorURL       string
	AuthorName      string
	BlogDescription string
}

func (rss *Rss) RssHandler(w http.ResponseWriter, r *http.Request) {
	feed := &feeds.Feed{
		Title:       rss.BlogName,
		Link:        &feeds.Link{Href: rss.BlogURL},
		Description: rss.BlogDescription,
		Author:      &feeds.Author{Name: rss.AuthorName, Email: rss.AuthorURL},
	}

	for _, post := range blogPosts {
		layout := "2006-01-02 15:04"
		editedTime, err := time.Parse(layout, post.EditedTS)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		layout = "2006-01-02"
		CreatedTime, err := time.Parse(layout, post.CreatedAt)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		item := &feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/blog/%s", rss.BlogURL, post.Slug)},
			Updated:     editedTime,
			Description: post.Description,
			Created:     CreatedTime,
		}
		feed.Items = append(feed.Items, item)
	}

	rssFeed, err := feed.ToRss()
	if err != nil {
		http.Error(w, "Could not generate feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, err = w.Write([]byte(rssFeed))
	if err != nil {
		http.Error(w, "Could not generate feed", http.StatusInternalServerError)
		return
	}
}
