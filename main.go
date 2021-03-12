package main

import (
	"bytes"
	"fmt"
	"github.com/xiatechs/markdown-to-confluence/confluence"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/parser/pageparser"
	"github.com/yuin/goldmark"
)

func main() {
	err := filepath.WalkDir("./", func(path string, info os.DirEntry, err error) error {
		if strings.Contains(path, "vendor") || strings.Contains(path, ".github") {
			return filepath.SkipDir
		}
		if strings.HasSuffix(info.Name(), ".md") {
			if err := parseContent(path); err != nil {
				log.Println(err)
			}
		}

		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}

func parseContent(filename string) error {
	r, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return err
	}

	cfm, err := pageparser.ParseFrontMatterAndContent(r)
	if err != nil {
		return err
	}

	log.Println(filename)
	log.Println(cfm.FrontMatter)

	if len(cfm.FrontMatter) == 0 {
		return fmt.Errorf("no frontmatter")
	}

	var buf bytes.Buffer

	if err := goldmark.Convert(cfm.Content, &buf); err != nil {
		return err
	}

	log.Println(buf.String())
	err = checkConfluenceFunc()
	if err != nil{
		return err
	}
	return nil
}


func checkConfluenceFunc() error{
	// todo: search confluence for filename
	// some logic to see if content is accurate
	// update or create logic
	// push new data to page
	fmt.Println("running find page function: ")
	a := confluence.NewAPIClient()
	a.FindPage()
	return nil
}
