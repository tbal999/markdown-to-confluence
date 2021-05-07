// Package node is to enable reading through a repo and create a tree of content on confluence
package node

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xiatechs/markdown-to-confluence/confluence"
	"github.com/xiatechs/markdown-to-confluence/markdown"
	"github.com/xiatechs/markdown-to-confluence/todo"
)

var (
	masterTitles  []string              // used to verify whether pages need to be deleted or not
	visual        = false               // set to true if you want a more verbose cmd line output
	rootDir       string                // will contain the root folderpath of the repo
	nodeAPIClient *confluence.APIClient // api client will be stored here
)

// Node struct enables creation of a page tree
type Node struct {
	index    int                     // each node will have index (for visual only - can be removed)
	id       int                     // when page is created, page ID will be stored here.
	isFolder bool                    // true if folder node, false if file/attachment node
	alive    bool                    // for tracking if the folder has any valid content within it asides more folders
	path     string                  // file / folderpath will be stored here
	root     *Node                   // the parent page node will be linked here
	branches []*Node                 // any children page nodes will be stored here (used to delete pages)
	children *confluence.PageResults // to store a snapshot of folder page & children pages (used to delete pages)

}

// visual method just to print the journey of the nodes being created (and for testing purposes)
func (node *Node) visual() {
	if visual {
		if node.root == nil || node.alive {
			node.printLive()
		} else {
			node.printDead()
		}
	}
}

func (node *Node) printLive() {
	var folderOrFile string

	if node.isFolder {
		folderOrFile = "folder"
	} else {
		folderOrFile = "file"
	}

	log.Printf("This is an alive %s node, page ID is %d", folderOrFile, node.id)

	if node.root != nil {
		log.Printf("Path: %s, Root path: %s", node.path, node.root.path)
	} else {
		log.Printf("Path: %s", node.path)
	}
}

func (node *Node) printDead() {
	log.Printf("This is a dead node")
	log.Printf("Path: %s", node.path)
}

// newNode - create a new node object
func newNode() *Node {
	node := Node{}
	return &node
}

// newPageResults - create a new confluence.PageResults object
func newPageResults() *confluence.PageResults {
	results := confluence.PageResults{}
	return &results
}

// Instantiate begins the generation of a tree of the repo for confluence
// and starts the whole process from the top/root node
func (node *Node) Instantiate(projectPath string, client *confluence.APIClient) bool {
	if isFolder(projectPath) {
		node.index = 1
		node.path = projectPath
		rootDir = strings.ReplaceAll(strings.ReplaceAll(projectPath, ".", ""), "/", "")

		nodeAPIClient = client

		node.generateMaster()

		node.generateTODOPage()

		return true
	}

	return false
}

func (node *Node) generateTODOPage() {
	todonode := Node{}
	todonode.root = node

	page := todo.GenerateTODO(rootDir)

	err := todonode.checkConfluencePages(page)
	if err != nil {
		log.Println(err)
	}
}

func (node *Node) generateTitles() (string, string) {
	const nestedDepth = 2

	fullDir := strings.ReplaceAll(node.path, ".", "")
	fullDir = strings.TrimPrefix(fullDir, "/")
	dirList := strings.Split(fullDir, "/")
	dir := dirList[len(dirList)-1]

	if len(dirList) > nestedDepth {
		dir += "-"
		dir += dirList[len(dirList)-2]
	}

	if node.root != nil {
		dir += "-"
		dir += rootDir
	}

	return dir, fullDir
}

// generateFolderPage method
// if called, this node is a master node for a folder which has content in it.
// if there are valid files within the folder, then this node will create a page
// for the folder & store any files in that folder on that page as attachments.
func (node *Node) generateFolderPage() {
	dir, fullDir := node.generateTitles()

	node.isFolder = true

	masterpagecontents := markdown.FileContents{
		MetaData: map[string]interface{}{
			"title": dir,
		},
		Body: []byte(`<p>Welcome to the '<b>` + dir + `</b>' folder of this Xiatech code repo.</p>
		<p>This folder full path in the repo is: ` + fullDir + `</p>
<p>You will find attachments/images for this folder via the ellipsis at the top right.</p>
<p>Any markdown or subfolders is available in children pages under this page.</p>`),
	}

	err := node.checkConfluencePages(&masterpagecontents)
	if err != nil {
		log.Println(err)
	}
}

// generateMaster method is to convert Node to a folder node / master node where we can append
// files and subfolders to the folder node as child pages.
// a subnode is created and that node is used to crawl through files in folder
func (node *Node) generateMaster() {
	// these constants are created to aid navigation of iterate method lower down
	const checking = true

	const processing = false

	const Folders = true

	const Files = false

	node.visual()

	subNode := newNode()
	subNode.index = node.index + 1
	subNode.path = node.path
	subNode.root = node
	subNode.children = newPageResults()
	node.branches = append(node.branches, subNode)

	thereAreValidFiles := subNode.iterate(checking, Files)
	if thereAreValidFiles {
		node.alive = true
		node.generateFolderPage()
		subNode.iterate(processing, Files)
		subNode.iterate(processing, Folders)
		subNode.visual()
	} else {
		subNode.iterate(processing, Folders)
	}
}

// iteratefiles method is to iterate through the files in a folder.
// if it finds a file it will begin processing that file via fileInDirectoryCheck method
func (node *Node) iterate(checking, folders bool) bool {
	var validFile bool
	// Go 1.15 method: err := filepath.Walk(node.path, func(fpath string, info os.FileInfo, err error) error {
	// Go 1.16 method: err := filepath.WalkDir(node.path, func(fpath string, info os.DirEntry, err error) error {
	err := filepath.WalkDir(node.path, func(fpath string, info os.DirEntry, err error) error {
		if sub(node.path, fpath) {
			validFile = node.fileInDirectoryCheck(fpath, checking, folders)
			if validFile {
				return io.EOF
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	return validFile
}

func (node *Node) fileInDirectoryCheck(fpath string, checking, folders bool) bool {
	if !folders {
		valid := node.checkIfMarkDown(fpath, checking) // for checking & processing markdown files / images etc
		if valid && checking {
			return true
		}
	} else {
		node.checkOtherFileTypes(fpath) // you can process other file types inside this method
	}

	return false
}

func (node *Node) checkIfMarkDown(fpath string, checking bool) bool {
	if !isFolder(fpath) {
		if ok := node.markDownChecker(checking, fpath); ok {
			return true
		}
	}

	return false
}

func (node *Node) checkOtherFileTypes(fpath string) {
	if !node.checkIfFolder(fpath) {
		node.checkIfGoFile(fpath)
		node.checkForImageOrPuml(fpath)
	}
}

func (node *Node) checkIfFolder(fpath string) bool {
	if isFolder(fpath) && !isVendorOrGit(fpath) {
		node.verifyCreateNode(fpath)
		return true
	}

	return false
}

func (node *Node) processGoFile(fpath string) error {
	contents, err := ioutil.ReadFile(filepath.Clean(fpath))
	if err != nil {
		return err
	}

	fullpath := strings.Replace(fpath, ".", "", 2)

	fullpath = strings.TrimPrefix(fullpath, "/")

	todo.ParseGo(contents, fullpath)

	return nil
}

// verifyCreateNode method is to create a new sub master node if there is a folder in the current dir
// but if the node is dead - the node will connect to the node above this node instead - skipping that empty folder
func (node *Node) verifyCreateNode(fpath string) {
	if node.path != fpath {
		subNode := newNode()
		subNode.path = fpath

		if node.alive {
			subNode.root = node.root
		} else {
			if node.root != nil {
				subNode.root = node.root.root
			} else {
				subNode.root = node.root
			}
		}

		subNode.index = node.index + 1
		node.branches = append(node.branches, subNode)
		subNode.generateMaster()
	}
}

// markDownChecker method is where we will create or update the page, or upload or update attachments
// this method is also used to check whether the node is alive or not
// a node is only considered alive if it has markdown files.
func (node *Node) markDownChecker(checkingOnly bool, path string) bool {
	markDownFilesExist := node.checkIfMarkDownFile(checkingOnly, path)

	if markDownFilesExist {
		node.alive = true
		return true
	}

	return false
}

// checkMarkDown method - check to see if the name of the file ends with .md i.e it's a markdown file
func (node *Node) checkIfGoFile(name string) {
	if strings.HasSuffix(name, ".go") {
		err := node.processGoFile(name)
		if err != nil {
			log.Println(err)
		}
	}
}

// checkMarkDown method - check to see if the name of the file ends with .md i.e it's a markdown file
func (node *Node) checkIfMarkDownFile(checking bool, name string) bool {
	if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".MD") {
		if !checking {
			err := node.processMarkDown(name)
			if err != nil {
				log.Println(err)
			}
		}

		return true
	}

	return false
}

// processFile is the method called on any eligible files (markdown / images etc) to handle uploads.
func (node *Node) processMarkDown(path string) error {
	contents, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}

	parsedContents, err := markdown.ParseMarkdown(node.root.id, contents)
	if err != nil {
		return err
	}

	err = node.checkConfluencePages(parsedContents)
	if err != nil {
		log.Printf("error completing confluence operations: %s", err)
	}

	return nil
}

// Scrub method clears away any pages on confluence that shouldn't exist
// this method should be called from the top node as it works top down
func (node *Node) Scrub() {
	if node.id != 0 {
		id := strconv.Itoa(node.id)
		node.findPagesToDelete(id)
	}

	for index := range node.branches {
		node.branches[index].Scrub()
	}
}

// findPagesToDelete method grabs results of page to begin deleting
func (node *Node) findPagesToDelete(id string) {
	children, err := nodeAPIClient.FindPage(id, true)
	if err != nil {
		log.Printf("error finding page: %s", err)
	}

	if children != nil {
		node.deletePages(children)
	}
}

// deletePages method is to find a page to delete
// and any children pages that might need to be deleted
func (node *Node) deletePages(children *confluence.PageResults) {
	for index := range children.Results {
		var noDelete bool

		for index2 := range masterTitles {
			if children.Results[index].Title == masterTitles[index2] {
				noDelete = true
				break
			}
		}

		if !noDelete {
			node.findPagesToDelete(children.Results[index].ID)
			node.deletePage(children.Results[index].ID)
		}
	}
}

// checkOtherFiles - check to see if the file is a puml or png image
func (node *Node) checkForImageOrPuml(name string) {
	if node.alive { // we only want to upload images that contained in same folder as markdown
		validFiles := []string{".puml", ".png", ".jpg", ".jpeg"}

		for index := range validFiles {
			if strings.Contains(name, validFiles[index]) {
				node.isNodeRootNil(name)
			}
		}
	}
}

// preUpload method to do some check(s) on a file before uploading them
func (node *Node) isNodeRootNil(name string) {
	if node.root != nil {
		node.uploadFile(name)
	}
}

// uploadFile is for uploading files to a specific page by root node page id
func (node *Node) uploadFile(path string) {
	if nodeAPIClient != nil {
		err := nodeAPIClient.UploadAttachment(filepath.Clean(path), node.root.id)
		if err != nil {
			log.Printf("error uploading attachment: %s", err)
		}
	}
}

// checkConfluencePages runs through the CRUD operations for confluence
func (node *Node) checkConfluencePages(newPageContents *markdown.FileContents) error {
	if nodeAPIClient == nil {
		return nil
	}

	pageTitle := strings.Join(strings.Split(newPageContents.MetaData["title"].(string), " "), "+")

	pageResult, err := nodeAPIClient.FindPage(pageTitle, false)
	if err != nil {
		return err
	}

	if pageResult == nil {
		err := node.generatePage(newPageContents)
		if err != nil {
			return err
		}
	} else {
		err = node.grabpagedata(*pageResult)
		if err != nil {
			return err
		}
		err = nodeAPIClient.UpdatePage(node.id, int64(pageResult.Results[0].Version.Number), newPageContents)
		if err != nil {
			return err
		}
	}

	masterTitles = append(masterTitles, newPageContents.MetaData["title"].(string))

	return nil
}

// grabpagedata method is to grab page ID and pass it to node
func (node *Node) grabpagedata(pageResult confluence.PageResults) error {
	var err error

	if len(pageResult.Results) > 0 {
		node.id, err = strconv.Atoi(pageResult.Results[0].ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// generatePage method is for validation to make sure client is not nil and node.root is not nil
func (node *Node) generatePage(newPageContents *markdown.FileContents) error {
	var err error

	if nodeAPIClient != nil {
		if node.root == nil {
			node.id, err = nodeAPIClient.CreatePage(0, newPageContents, true)
		} else {
			node.id, err = nodeAPIClient.CreatePage(node.root.id, newPageContents, false)
		}
	}

	return err
}

func isVendorOrGit(name string) bool {
	if strings.Contains(name, "vendor") || strings.Contains(name, ".github") || strings.Contains(name, ".git") {
		return true
	}

	return false
}

// isFolder checks whether a file is a folder or not
func isFolder(name string) bool {
	file, err := os.Open(filepath.Clean(name))
	if err != nil {
		log.Println(err)
		return false
	}

	defer func() {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Println(err)
		return false
	}

	if fileInfo.IsDir() {
		return true
	}

	return false
}

func (node *Node) deletePage(id string) {
	convert, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("error getting page ID: %s", err)
		return
	}

	err = nodeAPIClient.DeletePage(convert)
	if err != nil {
		log.Printf("error deleting page: %s", err)
	}
}

// checks to see if the file is within 1 level subdirectory of the base path
func sub(base, path string) bool {
	return strings.Count(path, "/")-strings.Count(base, "/") == 1
}
