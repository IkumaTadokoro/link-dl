package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FileLink struct {
	Name string
	URL  string
}

type Config struct {
	URL       string
	OutDir    string
	Parallel  int
	Exts      []string
	All       bool
	Include   string
	ListOnly  bool
	UserAgent string
}

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func main() {
	config := parseFlags()

	if config.URL == "" {
		printUsage()
		os.Exit(1)
	}

	// Extract links
	links, err := extractLinks(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting links: %v\n", err)
		os.Exit(1)
	}

	if len(links) == 0 {
		fmt.Println("No matching files found.")
		return
	}

	fmt.Printf("Found %d files:\n\n", len(links))
	for i, link := range links {
		fmt.Printf("  %3d. %s\n       %s\n", i+1, link.Name, link.URL)
	}
	fmt.Println()

	if config.ListOnly {
		return
	}

	// Create output directory
	if err := os.MkdirAll(config.OutDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Download files
	downloadAll(links, config)
}

func parseFlags() Config {
	outDir := flag.String("out", "./downloads", "Output directory")
	parallel := flag.Int("parallel", 5, "Number of parallel downloads")
	exts := flag.String("ext", "pdf,xlsx,xls,xlsm", "Comma-separated file extensions to download")
	all := flag.Bool("all", false, "Download all file links (ignores --ext)")
	include := flag.String("include", "", "Regex pattern to filter URLs")
	listOnly := flag.Bool("list", false, "List files only, don't download")
	userAgent := flag.String("ua", defaultUserAgent, "User-Agent header")

	flag.Usage = printUsage
	flag.Parse()

	var extensions []string
	if !*all && *exts != "" {
		for _, ext := range strings.Split(*exts, ",") {
			ext = strings.TrimSpace(ext)
			ext = strings.TrimPrefix(ext, ".")
			if ext != "" {
				extensions = append(extensions, "."+strings.ToLower(ext))
			}
		}
	}

	targetURL := ""
	if flag.NArg() > 0 {
		targetURL = flag.Arg(0)
	}

	return Config{
		URL:       targetURL,
		OutDir:    *outDir,
		Parallel:  *parallel,
		Exts:      extensions,
		All:       *all,
		Include:   *include,
		ListOnly:  *listOnly,
		UserAgent: *userAgent,
	}
}

func printUsage() {
	fmt.Println(`link-dl - Download files from any webpage

Usage:
  link-dl [options] <URL>

Examples:
  link-dl "https://example.com/documents"
  link-dl "https://example.com/page" --ext pdf,docx,zip
  link-dl "https://example.com/page" --all
  link-dl "https://example.com/page" --list
  link-dl "https://example.com/page" --include "2024.*\.pdf"

Options:`)
	flag.PrintDefaults()
}

func extractLinks(config Config) ([]FileLink, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", config.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	var includePattern *regexp.Regexp
	if config.Include != "" {
		includePattern, err = regexp.Compile(config.Include)
		if err != nil {
			return nil, fmt.Errorf("invalid include pattern: %v", err)
		}
	}

	var links []FileLink
	seen := make(map[string]bool)

	// Common file extensions for --all mode
	fileExtPattern := regexp.MustCompile(`(?i)\.(pdf|docx?|xlsx?|xlsm|pptx?|csv|txt|zip|rar|7z|tar|gz|jpg|jpeg|png|gif|svg|mp3|mp4|wav|avi|mov)$`)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		// Resolve relative URL
		linkURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}

		fullURL := linkURL.String()
		if seen[fullURL] {
			return
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(linkURL.Path))
		
		if config.All {
			// In --all mode, match common file extensions
			if !fileExtPattern.MatchString(linkURL.Path) {
				return
			}
		} else {
			// Check against specified extensions
			matched := false
			for _, e := range config.Exts {
				if ext == e {
					matched = true
					break
				}
			}
			if !matched {
				return
			}
		}

		// Check include pattern
		if includePattern != nil && !includePattern.MatchString(fullURL) {
			return
		}

		seen[fullURL] = true

		// Get link text as filename
		name := strings.TrimSpace(s.Text())
		if name == "" {
			// Fallback to URL filename
			name = filepath.Base(linkURL.Path)
		}

		// Sanitize filename
		name = sanitizeFilename(name)

		// Ensure correct extension
		if !strings.HasSuffix(strings.ToLower(name), ext) && ext != "" {
			name = name + ext
		}

		links = append(links, FileLink{Name: name, URL: fullURL})
	})

	return links, nil
}

func sanitizeFilename(name string) string {
	// Remove or replace invalid characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = invalid.ReplaceAllString(name, "_")

	// Replace multiple spaces/underscores
	name = regexp.MustCompile(`[\s_]+`).ReplaceAllString(name, "_")

	// Trim spaces, dots, and underscores
	name = strings.Trim(name, " ._")

	// Limit length
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}

	if name == "" {
		name = "unnamed"
	}

	return name
}

func downloadAll(links []FileLink, config Config) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, config.Parallel)

	// Track used filenames for deduplication
	var mu sync.Mutex
	usedNames := make(map[string]int)

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	successCount := 0
	failCount := 0

	for _, link := range links {
		wg.Add(1)
		go func(link FileLink) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Get unique filename
			mu.Lock()
			filename := getUniqueFilename(config.OutDir, link.Name, usedNames)
			mu.Unlock()

			fpath := filepath.Join(config.OutDir, filename)

			err := downloadFile(client, link.URL, fpath, config.UserAgent)
			mu.Lock()
			if err != nil {
				fmt.Printf("✗ %s: %v\n", filename, err)
				failCount++
			} else {
				fmt.Printf("✓ %s\n", filename)
				successCount++
			}
			mu.Unlock()
		}(link)
	}

	wg.Wait()
	fmt.Printf("\nDone! Success: %d, Failed: %d\n", successCount, failCount)
}

func getUniqueFilename(dir, name string, usedNames map[string]int) string {
	base := name
	ext := filepath.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, ext)

	count := usedNames[base]
	usedNames[base] = count + 1

	if count == 0 {
		// Check if file exists on disk
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			return name
		}
	}

	// Find next available number
	for i := count + 1; ; i++ {
		newName := fmt.Sprintf("%s_%d%s", nameWithoutExt, i, ext)
		if _, err := os.Stat(filepath.Join(dir, newName)); os.IsNotExist(err) {
			return newName
		}
	}
}

func downloadFile(client *http.Client, url, filepath, userAgent string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
