// Package swagger provides a method for working with and parsing swagger documents
//
//nolint:wsl // is fine
package swagger

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/gohugoio/hugo/parser/pageparser"
	"github.com/xiatechs/markdown-to-confluence/markdown"
)

// GrabAuthors - do we want to collect authors?
var (
	GrabAuthors bool
)

// grabtitle function collects the filename of a markdown file
// and returns it as a string
//
//nolint:deadcode,unused // not used anymore
func grabtitle(path string) string {
	return strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
}

// newFileContents function creates a new filecontents object
func newFileContents() *markdown.FileContents {
	f := markdown.FileContents{}
	f.MetaData = make(map[string]interface{})

	return &f
}

type author struct {
	name    string
	howmany int
}

type authors []author // not using a map so the order of authors can be maintained

func (a *authors) append(item string) {
	au := *a
	var exists bool
	for index := range au {
		if au[index].name == item {
			au[index].howmany++
			exists = true
			break
		}
	}

	if !exists {
		au = append(au, author{
			name:    item,
			howmany: 1,
		})
	}

	*a = au
}

func (a *authors) sort() {
	au := *a

	sort.Slice(au, func(i, j int) bool {
		return au[i].howmany > au[j].howmany
	})

	*a = au
}

// use git to capture authors by username & email & commits
//
//nolint:gosec // is fine
func capGit(path string) string {
	here, _ := os.Getwd()
	log.Println("collecting authorship for ", path)
	git := exec.Command("git", "log", "--all", `--format='%an | %ae'`, "--", here)

	out, err := git.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	a := authors{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		l := strings.ReplaceAll(line, `'`, "")
		a.append(l)
	}

	a.sort()

	// to let the output be displayed in confluence - wrapping it in code block
	output := `<pre><code>
[authors | email addresses | how many commits]
`

	index := 0
	for _, author := range a {
		if author.name == "" {
			continue
		}

		no := strconv.Itoa(author.howmany)

		if index == 0 {
			output += author.name + " - total commits: " + no
		} else {
			output += `
` + author.name + " - total commits: " + no
		}

		index++
	}

	output += `
</code></pre>`

	return output
}

// ParseSwagger function parses the swagger file and returns a filecontents object
func ParseSwagger(rootID int, content []byte, isIndex bool,
	pages map[string]string, path, abs, fileName string) (*markdown.FileContents, error) {
	r := bytes.NewReader(content)
	f := newFileContents()
	fmc, err := pageparser.ParseFrontMatterAndContent(r)
	if err != nil {
		log.Println("issue parsing frontmatter - (using # title instead): %w", err)
	} else if len(fmc.FrontMatter) != 0 {
		f.MetaData = fmc.FrontMatter
	}

	f.MetaData["title"] = fileName

	value, ok := f.MetaData["title"]
	if !ok {
		return nil, fmt.Errorf("swagger page parsing error - page title is not assigned via toml or # section")
	}

	if value == "" {
		return nil, fmt.Errorf("swagger page parsing error - page title is empty")
	}

	macroStart := `<ac:structured-macro ac:name="open-api" ac:schema-version="1" ac:macro-id="9819e58b-6a9a-4bd7-9503-385df5460d27"><ac:plain-text-body><![CDATA[`

	macroEnd := `]]></ac:plain-text-body></ac:structured-macro>`

	bodyString := macroStart + string(content) + macroEnd

	f.Body = []byte(bodyString)

	if GrabAuthors {
		f.Body = append(f.Body, []byte(capGit(path))...)
	}

	return f, nil
}
