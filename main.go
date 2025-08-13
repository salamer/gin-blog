package main

import (
	"html/template"
	"io/fs"
	"log"
	"os"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-yaml/yaml"
	"github.com/russross/blackfriday/v2"
)

// Metadata defines the structure of metadata in Markdown files.
type Metadata struct {
	Title   string   `yaml:"title"`   // Title of the post
	Date    string   `yaml:"date"`    // Date of the post
	Tags    []string `yaml:"tags"`    // Tags of the post
	Summary string   `yaml:"summary"` // Summary of the post
}

// Post represents a blog post.
type Post struct {
	Title      string        // Title of the post
	Content    template.HTML // HTML content of the post
	Summary    string        // Summary of the post
	Date       time.Time     // Date of the post
	DateString string        // Date of the post in string format
	Tags       []string      // Tags of the post
	Filename   string        // Filename of the post
	Slug       string        // Slug of the post
}

// getPosts retrieves all blog posts.
func getPosts() ([]Post, error) {
	contentDir := "content"
	var posts []Post

	// Read files in the content directory
	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Read the file content
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse metadata
		lines := strings.SplitN(string(data), "---", 3)
		if len(lines) < 3 {
			return nil
		}
		metaDataStr := lines[1]
		contentStr := lines[2]

		var metadata Metadata
		err = yaml.Unmarshal([]byte(metaDataStr), &metadata)
		if err != nil {
			return err
		}

		// Parse the date
		date, err := time.Parse("2006-01-02", metadata.Date)
		if err != nil {
			info, _ := d.Info()
			date = info.ModTime()
		}

		// Parse Markdown content
		output := blackfriday.Run([]byte(contentStr))

		post := Post{
			Title:      metadata.Title,
			Content:    template.HTML(output),
			Date:       date,                           // real date or file modification date
			DateString: date.Format("January 2, 2006"), // Format: January 2, 2006
			Tags:       metadata.Tags,
			Filename:   d.Name(),
			Summary:    metadata.Summary,
			Slug:       strings.ReplaceAll(strings.ToLower(metadata.Title), " ", "-"),
		}
		posts = append(posts, post)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort posts by date in descending order
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	return posts, nil
}

func main() {
	r := gin.Default()

	// Load templates
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	// Home page route
	r.GET("/", func(c *gin.Context) {
		posts, err := getPosts()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get posts"})
			return
		}
		c.HTML(200, "index.html", gin.H{
			"posts": posts,
		})
	})

    r.GET("/ping", func(c *gin.Context) {
        clientIP := c.ClientIP()
        fmt.Println("Request from IP:", clientIP)
        c.String(http.StatusOK, "pong")
    })

	// Single post route
	r.GET("/blog/:postTitle", func(c *gin.Context) {
		postTitle := c.Param("postTitle")
		posts, err := getPosts()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get posts"})
			return
		}

		var foundPost Post
		for _, post := range posts {
			if post.Slug == postTitle {
				foundPost = post
				break
			}
		}

		if foundPost.Title == "" {
			c.JSON(404, gin.H{"error": "Post not found"})
			return
		}

		c.HTML(200, "single.html", gin.H{
			"post": foundPost,
		})
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
