package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

type BlogPost struct {
	Title       string
	Slug        string
	EditedTS    string
	CreatedAt   string
	Description string
	HTMLContent template.HTML
	Post        bool
	Draft       bool
}

type BlogList struct {
	Post  bool
	Title string
	Posts []BlogPost
}

type FrontMatter struct {
	CreatedAt   time.Time `yaml:"created"`
	Description string    `yaml:"description"`
	Draft       bool      `yaml:"draft"`
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

func convertMarkdownToHTML(input string) (string, error) {
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

func parseFrontMatter(content string) (FrontMatter, string, error) {
	var meta FrontMatter

	// Require the first "---" to be at the very start
	if !strings.HasPrefix(content, "---") {
		return meta, content, nil
	}

	parts := strings.SplitN(content, "---", 3)

	if len(parts) < 3 {
		return meta, content, nil
	}

	err := yaml.Unmarshal([]byte(parts[1]), &meta)
	if err != nil {
		return meta, "", err
	}

	return meta, parts[2], nil
}

func loadBlogPosts(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	newPosts := make(map[string]BlogPost)

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			fullPath := filepath.Join(path, f.Name())
			rawContent, err := os.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("could not read file %s: %w", f.Name(), err)
			}

			// Parse Front Matter and Content
			meta, markdownOnly, err := parseFrontMatter(string(rawContent))
			if err != nil {
				logger.Warn(fmt.Sprintf("Skipping %s: invalid front matter", f.Name()))
				continue
			}

			// Check for Draft status
			if meta.Draft {
				continue
			}

			html, err := convertMarkdownToHTML(markdownOnly)
			if err != nil {
				return err
			}

			modedTime, _ := fileModed(fullPath)
			name := strings.TrimSuffix(f.Name(), ".md")

			newPosts[name] = BlogPost{
				Title:       name,
				Slug:        name,
				EditedTS:    modedTime,
				CreatedAt:   meta.CreatedAt.String()[:10],
				Description: meta.Description,
				HTMLContent: template.HTML(html),
				Post:        true,
			}
		}
	}

	blogMux.Lock()
	blogPosts = newPosts
	blogMux.Unlock()

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
		return allPosts[i].CreatedAt > allPosts[j].CreatedAt
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
