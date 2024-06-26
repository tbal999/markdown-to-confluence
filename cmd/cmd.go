// Package cmd contains code necessary to start the github action
// taking in arguments & setting them in variables in common package
package cmd

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/xiatechs/markdown-to-confluence/common"
	"github.com/xiatechs/markdown-to-confluence/confluence"
	"github.com/xiatechs/markdown-to-confluence/markdown"
	"github.com/xiatechs/markdown-to-confluence/node"
)

// setArgs function takes in cmd line arguments
// and sets common variables (api key / space / username / project path / master page ID / confluenceURL / only docs)
func setArgs() bool {
	var argLength = 7

	if len(os.Args) < argLength-1 {
		log.Println("usage: apikey space repopath masterpageID confluenceURL onlyDocs")
		return false
	}

	vars := os.Args[1:]

	var err error

	if len(vars) == argLength-1 {
		common.ConfluenceAPIKey = vars[0]
		common.ConfluenceSpace = vars[1]

		common.ProjectPathEnv = vars[2]
		common.ProjectPathEnv = strings.ReplaceAll(common.ProjectPathEnv, " ", "-") // replace spaces with -

		common.ProjectMasterID, err = strconv.Atoi(vars[3])
		if err != nil {
			log.Println("masterpageID should be an int. If mtc is to be the root enter 0")
			return false
		}

		if vars[4] != "" {
			common.ConfluenceBaseURL = vars[4]
		}

		common.OnlyDocs, err = strconv.ParseBool(vars[5])
		if err != nil {
			log.Println("onlyDocs should be a bool")
			return false
		}
	}

	return true
}

// Start function sets argument inputs, creates confluence API client
// and begins the process of creating confluence pages via calling
// the node.Start method
// if node.Start returns true, then calls node.Delete method
// returns the exit code for the program
func Start() int {
	markdown.GrabAuthors = false

	if setArgs() {
		root := node.Node{}

		client, err := confluence.CreateAPIClient()
		if err != nil {
			log.Println(err)
			return 1
		}

		node.SetAPIClient(client)

		if root.Start(common.ProjectMasterID, common.ProjectPathEnv, common.OnlyDocs) {
			root.Delete()
			return 0
		}
	}

	return 1
}
