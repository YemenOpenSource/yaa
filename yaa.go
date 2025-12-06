package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	yaasearch "yaa/yaasearch"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Yaa",
		Usage: "Yaml Search for Humans",

		Commands: []*cli.Command{
			{
				Name:      "search",
				Aliases:   []string{"s"},
				UsageText: "Yaa search [options] <query...>",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Value:   10,
						Usage:   "Number of results to display",
					},
					&cli.StringFlag{
						Name:    "export",
						Aliases: []string{"e"},
						Usage:   "Path to save yaml files",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Overwrite existing files when exporting",
					},
					&cli.BoolFlag{
						Name:    "debug",
						Aliases: []string{"d"},
						Usage:   "Enable verbose debug logging",
					},
				},

				Action: searchAction,
			},
			{
				Name:    "index",
				Aliases: []string{"i"},
				Usage:   "Yaa index [options] <folder>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "debug",
						Aliases: []string{"d"},
						Usage:   "Enable verbose debug logging",
					},
				},
				Action: indexAction,
			},
		},
	}

	// configure default logger
	log.SetFlags(log.LstdFlags)
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
}

func searchAction(c *cli.Context) error {
	// Set debug level for search run
	yaasearch.Debug = c.Bool("debug")
	if yaasearch.Debug {
		log.Println("Debug logging enabled")
	}
	query := c.Args().Slice()
	if len(query) == 0 {

		return cli.Exit("No query was found, use -h for help.", 1)
	}

	limit := c.Int("limit")
	results := yaasearch.Search(query, limit)
	if results == nil {
		log.Println("Search failed or index missing")
		return cli.Exit("Search failed or index missing", 1)
	}

	// Bleve's Hits is a collection; use Len()
	if results.Hits.Len() > 0 {
		if c.IsSet("export") {
			dest_path := c.String("export")
			// Ensure export directory exists and is a directory
			if err := os.MkdirAll(dest_path, 0o755); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fi, err := os.Stat(dest_path)
			if err != nil || !fi.IsDir() {
				return cli.Exit("Export path must be a directory", 1)
			}
			// ID is set to the file path
			force := c.Bool("force")
			for _, hit := range results.Hits {
				if err := exportFile(hit.ID, dest_path, force); err != nil {
					log.Println("Export error:", err)
				}
			}
			log.Println(len(results.Hits), "files exported to", dest_path)
			return nil
		}
		log.Println(results)
	} else {

		log.Println("No Match Found")
	}
	return nil
}

func indexAction(c *cli.Context) error {

	path := c.Args().First()
	if path == "" {
		return cli.Exit("Please provide a folder to index", 1)
	}
	yaasearch.Debug = c.Bool("debug")
	if yaasearch.Debug {
		log.Println("Debug logging enabled")
	}
	yaasearch.Index(path)
	return nil
}

// copy the file to destination path specified in the export option
func exportFile(srcFilePath, destPath string, force bool) error {
	filename := filepath.Base(srcFilePath)
	dest_file := filepath.Join(destPath, filename)
	// Prevent overwriting unless force is set
	if !force {
		if _, err := os.Stat(dest_file); err == nil {
			return fmt.Errorf("destination file exists: %s", dest_file)
		}
	}
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	// Use O_EXCL when not forcing to avoid race overwrites
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	destFile, err := os.OpenFile(dest_file, flags, 0o644)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	err = destFile.Sync()
	if err != nil {
		return err
	}
	return nil
}
