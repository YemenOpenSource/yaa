package yaasearch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/highlight/highlighter/ansi"
	"gopkg.in/yaml.v3"
)

var indexDir = "yaml_index"

// Debug controls verbose logging
var Debug bool

func info(v ...any) {
	log.Println(v...)
}

func debug(v ...any) {
	if Debug {
		log.Println(v...)
	}
}

func Index(dataDir string) error {

	// Open or create a new index
	index, err := bleve.Open(indexDir)
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexDir, mapping)
		if err != nil {
			info("Error creating index:", err)
			return err
		}
	} else if err != nil {
		info("Error opening index:", err)
		return err
	}

	stopChan := make(chan struct{})
	go showIndicatorsDots(stopChan)
	// Walk through the YAML files and index them
	err = filepath.Walk(dataDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".yml") || strings.HasSuffix(strings.ToLower(path), ".yaml")) {
			data, err := os.ReadFile(path)
			if err != nil {
				info("Error reading file", path, ":", err)
				return nil
			}
			// Parse the YAML data
			var yamlData map[string]interface{}

			if err := yaml.Unmarshal(data, &yamlData); err != nil {
				info("Error parsing YAML file", path, ":", err)
				return nil
			}

			if err := index.Index(path, yamlData); err != nil {
				info("Error indexing file", path, ":", err)
				return err
			}
			debug("Indexed", path)
		}
		return nil
	})

	if err != nil {
		info("Error walking the directory:", err)
		return err
	}
	close(stopChan)
	info("Indexing Done!")

	index.Close()
	return nil
}

func Search(query []string, limit int) *bleve.SearchResult {
	// Search for a term within the index
	if indexExists(indexDir) {
		index, err := bleve.Open(indexDir)
		if err != nil {
			info("Error searching index:", err)
			return nil
		}
		defer index.Close()

		queryStr := strings.Join(query, " ")
		q := bleve.NewQueryStringQuery(queryStr)
		search := bleve.NewSearchRequest(q)
		search.Size = limit
		search.Highlight = bleve.NewHighlightWithStyle(ansi.Name)
		result, err := index.Search(search)

		if err != nil {
			info("Error searching index:", err)
			return nil
		}

		debug("Search query:", queryStr, "hits:", result.Hits.Len())
		return result
	} else {
		info("Index was not found")
		return nil
	}
}

func indexExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false // Folder does not exist
	}
	return false
}

func showIndicatorsDots(stopChan <-chan struct{}) {
	dots := []string{".", "..", "...", "...."}
	index := 0

	for {
		select {
		case <-stopChan:
			return
		default:
			fmt.Printf("\rIndexing%s", dots[index])
			index = (index + 1) % len(dots)
			time.Sleep(500 * time.Millisecond)
		}
	}
}
