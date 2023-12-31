package git

import (
	"encoding/json"
	"fmt"
	"github-backup/pkg/metrics"
	"github.com/rs/zerolog/log"
	"github.com/tomnomnom/linkheader"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Repo struct {
	FullName string `json:"full_name"`
	PushedAt string `json:"pushed_at"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

var client = http.Client{}

func ReposFor(org, authToken string) ([]Repo, error) {
	m := metrics.RepoCount.WithLabelValues(org)

	urlRaw := fmt.Sprintf("https://api.github.com/orgs/%s/repos", org)
	var allRepos []Repo
	for urlRaw != "" {
		log.Info().Msgf("Retrieving repos from %s", urlRaw)
		var reposPart []Repo
		res, err := getRequest(urlRaw, authToken)
		if res.StatusCode != 200 {
			log.Error().Msgf("got a %d from GitHub", res.StatusCode)
			return nil, err
		}
		if err != nil {
			log.Error().Msgf("Request to '%s' filed: %v", urlRaw, err)
			return nil, err
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error().Msgf("Error reading data from '%s': %v", urlRaw, err)
			return nil, err
		}
		err = json.Unmarshal(body, &reposPart)
		if err != nil {
			log.Error().Msgf("Error unmarshaling data: ", err)
			return nil, err
		}
		allRepos = append(allRepos, reposPart...)
		m.Add(float64(len(reposPart)))
		linkHeader := res.Header.Get("Link")
		urlRaw = nextUrl(linkHeader)
	}
	return allRepos, nil
}

func getRequest(rawUrl, authToken string) (*http.Response, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		log.Error().Msgf("Parsing url failed: %v", rawUrl)
		return nil, err
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Error().Msgf("Could not GET '%s': %v", rawUrl, err)
		return nil, err
	}
	req.Header = http.Header{
		"Accept":        {"application/vnd.github.v3+json"},
		"User-Agent":    {"Backup"},
		"Authorization": {fmt.Sprintf("Bearer %s", authToken)},
	}
	res, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Error in  Client.Do: %v", err)
		return nil, err
	}
	return res, nil
}

func nextUrl(linkHeader string) string {
	links := linkheader.Parse(linkHeader)
	for _, link := range links {
		if link.Rel == "next" {
			return link.URL
		}
	}
	return ""
}
