package main

import (
	"fmt"
	"os"

	"github.com/engelsjk/criticalityscore/criticalityscore"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app     = kingpin.New("criticalityscore", "gives criticality score for an open source project")
	repoURL = app.Flag("repo", "repository url").Required().String()
	format  = app.Flag("format", "output format. allowed values are [default, csv, json]").Default("default").String()
	params  = app.Flag("param", "additional parameter in form <value>:<weight>:<max_threshold>").Strings()
)

func main() {

	app.Version("0.0.1")
	_, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		fmt.Println("warning: env variable GITHUB_AUTH_TOKEN not provided")
	}

	repo, err := criticalityscore.LoadRepository(*repoURL, token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	score, err := criticalityscore.RepositoryStats(repo, *params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	criticalityscore.PrintScore(score, *format)
}
