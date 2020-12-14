// # Copyright 2020 Jon Engelsman
// # Copyright 2020 Google LLC
// #
// # Licensed under the Apache License, Version 2.0 (the "License");
// # you may not use this file except in compliance with the License.
// # You may obtain a copy of the License at
// #
// #      http://www.apache.org/licenses/LICENSE-2.0
// #
// # Unless required by applicable law or agreed to in writing, software
// # distributed under the License is distributed on an "AS IS" BASIS,
// # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// # See the License for the specific language governing permissions and
// # limitations under the License.

// todo: add additional param parse/validator from value:weight:threshold args

package criticalityscore

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
)

var (
	ErrUnknownOutputFormat error = fmt.Errorf("unknown output format")
	ErrInvalidParamFormat  error = fmt.Errorf("invalid param format")
)

type Score struct {
	Name                string  `json:"name"`
	URL                 string  `json:"url"`
	Language            string  `json:"language"`
	CreatedSince        int     `json:"created_since"`
	UpdatedSince        int     `json:"updated_since"`
	ContributorCount    int     `json:"contributor_count"`
	OrgCount            int     `json:"org_count"`
	CommitFrequency     float64 `json:"commit_frequency"`
	RecentReleasesCount int     `json:"recent_releases_count"`
	ClosedIssuesCount   int     `json:"closed_issues_count"`
	UpdatedIssuesCount  int     `json:"updated_issues_count"`
	CommentFrequency    float64 `json:"comment_frequency"`
	DependentsCount     int     `json:"dependents_count"`
	CriticalityScore    float64 `json:"criticality_score"`
	ScoredOn            string  `json:"scored_on"`
}

func ParamScore(param interface{}, maxValue, weight float64) float64 {
	var p float64
	switch v := param.(type) {
	case float64:
		p = v
	case int:
		p = float64(v)
	}
	return math.Log(1.0+p) / math.Log(1.0+math.Max(p, maxValue)) * weight
}

type AdditionalParam struct {
	Value        float64
	Weight       float64
	MaxThreshold float64
}

func RepositoryStats(ghr GitHubRepository, params []string) (Score, error) {

	additionalParams, err := parseAdditionalParams(params)
	if err != nil {
		return Score{}, fmt.Errorf("%s : %s", ErrInvalidParamFormat.Error(), err.Error())
	}

	additionalParamsTotalWeight := 0.0
	additionalParamsScore := 0.0

	for _, param := range additionalParams {
		additionalParamsTotalWeight += param.Weight
		additionalParamsScore += ParamScore(param.Value, param.MaxThreshold, param.Weight)
	}

	score := Score{
		Name:     ghr.R.GetName(),
		URL:      ghr.R.GetHTMLURL(),
		Language: ghr.R.GetLanguage(),
	}

	wg := new(sync.WaitGroup)
	wg.Add(9)

	go func() {
		score.CreatedSince = ghr.CreatedSince()
		wg.Done()
	}()

	go func() {
		score.UpdatedSince = ghr.UpdatedSince()
		wg.Done()
	}()

	go func() {
		score.ContributorCount = ghr.Contributors()
		wg.Done()
	}()

	go func() {
		score.OrgCount = len(ghr.ContributorOrgs())
		wg.Done()
	}()

	go func() {
		score.CommitFrequency = ghr.CommitFrequency()
		wg.Done()
	}()

	go func() {
		score.RecentReleasesCount = ghr.RecentReleases()
		wg.Done()
	}()

	go func() {
		score.ClosedIssuesCount = ghr.ClosedIssues()
		wg.Done()
	}()

	go func() {
		score.UpdatedIssuesCount = ghr.UpdatedIssues()
		score.CommentFrequency = ghr.CommentFrequency(score.UpdatedIssuesCount)
		wg.Done()
	}()

	go func() {
		score.DependentsCount = ghr.Dependents()
		wg.Done()
	}()

	wg.Wait()

	if ghr.Error != nil {
		return Score{}, ghr.Error
	}

	totalWeight := CreatedSinceWeight + UpdatedSinceWeight +
		ContributorCountWeight + OrgCountWeight +
		CommitFrequencyWeight + RecentReleasesWeight +
		ClosedIssuesWeight + UpdatedIssuesWeight +
		CommentFrequencyWeight + DependentsCountWeight +
		additionalParamsTotalWeight

	score.CriticalityScore = math.Round((ParamScore(score.CreatedSince, CreatedSinceThreshold, CreatedSinceWeight)+
		ParamScore(score.UpdatedSince, UpdatedSinceThreshold, UpdatedSinceWeight)+
		ParamScore(score.ContributorCount, ContributorCountThreshold, ContributorCountWeight)+
		ParamScore(score.OrgCount, OrgCountThreshold, OrgCountWeight)+
		ParamScore(score.CommitFrequency, CommitFrequencyThreshold, CommitFrequencyWeight)+
		ParamScore(score.RecentReleasesCount, RecentReleasesThreshold, RecentReleasesWeight)+
		ParamScore(score.ClosedIssuesCount, ClosedIssuesThreshold, ClosedIssuesWeight)+
		ParamScore(score.UpdatedIssuesCount, UpdatedIssuesThreshold, UpdatedIssuesWeight)+
		ParamScore(score.CommentFrequency, CommentFrequencyThreshold, CommentFrequencyWeight)+
		ParamScore(score.DependentsCount, DependentsCountThreshold, DependentsCountWeight)+
		additionalParamsScore)/totalWeight*100000) / 100000

	score.ScoredOn = time.Now().UTC().Format(time.UnixDate)

	return score, nil
}

// PrintScore outputs all score values in the specified format (default, json or csv)
func PrintScore(score Score, format string) {

	if format == "default" {
		v := reflect.ValueOf(score)
		typeOfScore := v.Type()
		for i := 0; i < v.NumField(); i++ {
			fmt.Printf("%s: %v\n", typeOfScore.Field(i).Tag.Get("json"), v.Field(i).Interface())
		}
		return
	}

	if format == "csv" {
		w := csv.NewWriter(os.Stdout)
		v := reflect.ValueOf(score)
		typeOfScore := v.Type()
		for i := 0; i < v.NumField(); i++ {
			c1 := typeOfScore.Field(i).Tag.Get("json")
			var c2 string
			switch vv := v.Field(i).Interface().(type) {
			case string:
				c2 = vv
			case int:
				c2 = strconv.Itoa(vv)
			case float64:
				c2 = fmt.Sprintf("%0.1f", vv)
				if c1 == "CriticalityScore" {
					c2 = fmt.Sprintf("%0.5f", vv)
				}
			}
			line := []string{c1, c2}
			if err := w.Write(line); err != nil {
				log.Println(err.Error())
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			log.Println(err.Error())
		}
		return
	}

	if format == "json" {
		b, err := json.MarshalIndent(score, "", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
		return
	}

	fmt.Println(ErrUnknownOutputFormat.Error())
}
