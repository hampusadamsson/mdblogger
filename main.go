package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := setupFlags()

	// Logic to start the blog platform would go here
	logger.Info("Starting Blog Platform...")
	logger.Info(fmt.Sprintf("Folder=%s", cfg.ContentPath))
	logger.Info(fmt.Sprintf("Host=%s", cfg.Host))
	logger.Info(fmt.Sprintf("Port=%d", cfg.Port))
	// logger.Info(fmt.Sprintf("Watcher=%d", cfg.WatcherEnabled))
	logger.Info(fmt.Sprintf("Template=%s", cfg.TemplatePath))

	if cfg.WatcherEnabled {
		watchContentDir(cfg.ContentPath)
	}
	r := chi.NewRouter()
	logger.Debug("Router started...")

	loadTemplates(cfg.TemplatePath)
	logger.Debug("Templates loaded...")

	if err := loadBlogPosts(cfg.ContentPath); err != nil {
		logger.Error(err.Error())
	}
	logger.Debug("Blog posts loaded...")

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	logger.Debug("Statics loaded...")

	r.Handle("/blog/assets/*", http.StripPrefix("/blog/assets/", http.FileServer(http.Dir(cfg.ContentPath+"/assets"))))
	logger.Debug("Assets loaded...")

	r.Get("/blog/{slug}", blogHandler)
	logger.Debug("Blog slugs loaded...")

	r.Get("/", blogListHandler)
	logger.Debug("Root path loaded...")

	rss := Rss{BlogName: cfg.BlogName, BlogURL: cfg.BlogURL, BlogDescription: cfg.BlogDescription, AuthorName: cfg.AuthorName, AuthorEmail: cfg.AuthorEmail}
	r.Get("/rss.xml", rss.RssHandler)
	logger.Debug("RSS handler loaded...")

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), r)
	if err != nil {
		logger.Error(err.Error())
	}
}
