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

func (this *Rss) RssHandler(w http.ResponseWriter, r *http.Request) {
	feed := &feeds.Feed{
		Title:       this.BlogName,
		Link:        &feeds.Link{Href: this.BlogURL},
		Description: this.BlogDescription,
		Author:      &feeds.Author{Name: this.AuthorName, Email: this.AuthorURL},
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
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/blog/%s", this.BlogURL, post.Slug)},
			Updated:     editedTime,
			Description: post.Description,
			Created:     CreatedTime,
		}
		feed.Items = append(feed.Items, item)
	}

	rss, err := feed.ToRss()
	if err != nil {
		http.Error(w, "Could not generate feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(rss))
}
