package main

import (
	"log"
	"os"

	"github.com/engelsjk/criticalityscore"
)

func main() {

	r := "https://github.com/kubernetes/kubernetes"

	token := os.Getenv("GITHUB_AUTH_TOKEN")

	repo, err := criticalityscore.LoadRepository(r, token)
	if err != nil {
		log.Println(err.Error())
	}

	score, err := criticalityscore.RepositoryStats(repo, nil)
	if err != nil {
		log.Println(err.Error())
	}

	criticalityscore.PrintScore(score)
}
