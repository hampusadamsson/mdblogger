package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type BlogPost struct {
	Post        bool
	Title       string
	Slug        string
	EditedTS    string
	HTMLContent template.HTML
}

type BlogList struct {
	Post  bool
	Title string
	Posts []BlogPost
}

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.NewLinkify(),
		extension.Table,
		extension.Strikethrough,
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

func fileModed(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	return info.ModTime().String()[:16], nil
}

var (
	obsidianImageRe = regexp.MustCompile(`!\[\[(.*?)\]\]`)
	obsidianLinkRe  = regexp.MustCompile(`\[\[(.*?)\]\]`)
)

func ConvertObsidianImageLinks(input string) string {
	// Convert image links first: ![[file]] → ![img](assets/file)
	result := obsidianImageRe.ReplaceAllStringFunc(input, func(match string) string {
		g := obsidianImageRe.FindStringSubmatch(match)
		if len(g) < 2 {
			return match
		}
		encoded := url.PathEscape(g[1])
		return "![img](assets/" + encoded + ")"
	})

	// Convert normal links: [[file]] → [file](file)
	result = obsidianLinkRe.ReplaceAllStringFunc(result, func(match string) string {
		g := obsidianLinkRe.FindStringSubmatch(match)
		if len(g) < 2 {
			return match
		}
		encoded := url.PathEscape(g[1])
		return "[" + g[1] + "](" + encoded + ")"
	})

	return result
}

func markdownToHTML(path string) (string, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	inputLinksFixed := []byte(ConvertObsidianImageLinks(string(input)))

	var buf bytes.Buffer
	if err := md.Convert(inputLinksFixed, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var (
	blogPosts map[string]BlogPost
	blogMux   sync.RWMutex
)

var tpl *template.Template

func loadTemplates(folder string) {
	tpl = template.Must(template.ParseFiles(
		folder+"/layout.html",
		folder+"/blog_entry.html",
		folder+"/blog_list.html",
	))
}

func loadBlogPosts(path string) error {
	blogMux.Lock()
	defer blogMux.Unlock()

	posts := make(map[string]BlogPost)

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") {

			html, err := markdownToHTML(path + "/" + f.Name())
			if err != nil {
				return err
			}

			modedTime, err := fileModed(path + "/" + f.Name())
			if err != nil {
				return err
			}

			name := strings.TrimSuffix(f.Name(), ".md")

			posts[name] = BlogPost{
				Title:       strings.Title(strings.ReplaceAll(name, "-", " ")),
				Post:        true,
				Slug:        name,
				EditedTS:    modedTime,
				HTMLContent: template.HTML(html),
			}
		}
	}

	blogPosts = posts
	logger.Info(fmt.Sprintf("Found: %d posts\n", len(blogPosts)))
	return nil
}

func reloadBlogPosts(path string) {
	logger.Info("[watcher] Reloading markdown files...")
	if err := loadBlogPosts(path); err != nil {
		logger.Info(fmt.Sprintf("[watcher] Error reloading posts: %e", err))
	}
}

func watchContentDir(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(fmt.Sprintf("%s : %e", path, err))
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// relevant file operations
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					if strings.HasSuffix(event.Name, ".md") {
						log.Printf("[watcher] Detected change: %s", event)
						reloadBlogPosts(path)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Info(fmt.Sprintf("[watcher] error: %e", err))
			}
		}
	}()

	// Add path to watcher
	err = watcher.Add(path)
	if err != nil {
		logger.Error(err.Error())
	}

	logger.Debug("[watcher] Watching content/ for changes...")
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "slug")

	blogMux.RLock()
	post, ok := blogPosts[name]
	blogMux.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	err := tpl.ExecuteTemplate(w, "layout.html", post)
	if err != nil {
		logger.Error(err.Error())
	}
}

func blogListHandler(w http.ResponseWriter, r *http.Request) {
	blogMux.RLock()
	defer blogMux.RUnlock()

	// collect posts into a slice
	allPosts := make([]BlogPost, 0, len(blogPosts))
	for _, post := range blogPosts {
		allPosts = append(allPosts, post)
	}

	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].EditedTS > allPosts[j].EditedTS
	})

	data := BlogList{
		Posts: allPosts,
		Title: "list",
	}
	err := tpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		logger.Error(err.Error())
	}
}
