package main

import (
	"cloud.google.com/go/storage"
	"context"
	"github-backup/pkg/git"
	"github-backup/pkg/objstorage"
	"github-backup/pkg/zippings"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var basedir = filepath.Join(os.TempDir(), "ghbackup")

const MaxConcurrent = 10

func main() {

	var bucketname string
	var goog *storage.Client

	userName := envOrDie("GITHUB_USER")

	orgsstring := envOrDie("ORG_NAMES")
	orgs := strings.Split(orgsstring, ",")
	log.Info().Msgf("Using orgs '%s'", orgsstring)

	nfsShare, nfsStorage := os.LookupEnv("NFS_SHARE")

	if !nfsStorage {
		bucketname = envOrDie("BUCKET_NAME")
	} else {
		//Cleanup from failed runs.
		for _, org := range orgs {
			str := []string{nfsShare, org}
			path := strings.Join(str, "/")
			os.RemoveAll(path)
		}
		log.Info().Msgf("Using NFS storage on with directory '%s'", nfsShare)
	}

	githubToken := envOrDie("GITHUB_TOKEN")

	var repos []git.Repo
	for _, org := range orgs {
		repos = append(repos, reposOrDie(org, githubToken)...)
	}
	log.Info().Msgf("found %d repos", len(repos))

	if !nfsStorage {
		goog, err := storage.NewClient(context.Background())
		defer goog.Close()
		if err != nil {
			log.Error().Msgf("unable to create gcs client: %v", err)
			os.Exit(1)
		}
	}

	workQueue := make(chan int, MaxConcurrent)
	var wg sync.WaitGroup
	wg.Add(len(repos))
	for i, repo := range repos {
		log.Info().Msgf("processing repo %s - %d/%d", repo, i+1, len(repos))
		r := repo
		workQueue <- 1
		go func() {
			if !nfsStorage {
				err := cloneZipAndStoreInBucket(r.FullName, bucketname, userName, githubToken, goog)
				if err != nil {
					log.Error().Msgf("failed to backup repo '%s': %v", r.FullName, err)
				}
			} else {
				err := cloneZipAndStoreInFile(r.FullName, nfsShare, userName, githubToken)
				if err != nil {
					log.Error().Msgf("failed to backup repo '%s': %v", r.FullName, err)
				}
			}
			<-workQueue
			wg.Done()
		}()
	}
	wg.Wait()
}

func cloneZipAndStoreInFile(repo string, nfsShare string, userName string, githubToken string) error {
	compressedFileName := zippings.FilenameFor(repo)
	compressedFilePath := filepath.Join(nfsShare, compressedFileName)
	repodir := filepath.Join(nfsShare, repo)

	err := git.CloneRepo(nfsShare, repo, userName, githubToken)
	if err != nil {
		rm([]string{repodir})
		log.Error().Msgf("Failed while cloning repo '%s'", repo)
		return err
	}

	err = zippings.CompressIt(repodir, compressedFilePath)
	if err != nil {
		rm([]string{repodir, compressedFilePath})
		log.Error().Msgf("Error compressing repo '%s': '%v'", repo, err)
	}

	file, err := os.Open(compressedFilePath)
	defer file.Close()
	if err != nil {
		rm([]string{repodir, compressedFilePath})
		return err
	}

	err = os.RemoveAll(repodir)
	if err != nil {
		log.Error().Msgf("Error removing folder: '%a'", repodir)
	}
	return nil
}
func cloneZipAndStoreInBucket(repo string, bucketname string, userName string, githubToken string, gcsClient *storage.Client) error {
	compressedFileName := zippings.FilenameFor(repo)
	compressedFilePath := filepath.Join(basedir, compressedFileName)
	repodir := filepath.Join(basedir, repo)

	err := git.CloneRepo(basedir, repo, userName, githubToken)
	if err != nil {
		rm([]string{repodir})
		return err
	}

	err = zippings.CompressIt(repodir, compressedFilePath)
	if err != nil {
		rm([]string{repodir, compressedFilePath})
		return err
	}

	file, err := os.Open(compressedFilePath)
	defer file.Close()
	if err != nil {
		rm([]string{repodir, compressedFilePath})
		return err
	}

	if err != nil {
		panic(err)
	}
	objBasePath := time.Now().Format("2006/01/02")
	err = objstorage.CopyToBucket(gcsClient, file, bucketname, objBasePath)
	if err != nil {
		rm([]string{repodir, compressedFilePath})
		return err
	}

	rm([]string{repodir, compressedFilePath})

	return nil
}

func envOrDie(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		log.Error().Msgf("unable to find env var '%s', I'm useless without it", name)
		os.Exit(1)
	}
	return value
}

func reposOrDie(org string, githubToken string) []git.Repo {
	repos, err := git.ReposFor(org, githubToken)
	if err != nil {
		log.Error().Msgf("couldn't get list of repos: %v", err)
		os.Exit(1)
	}
	return repos
}

func rm(entries []string) {
	for _, f := range entries {
		log.Info().Msgf("deleting %s", f)
		err := os.RemoveAll(f)
		if err != nil {
			log.Error().Msgf("Unable to delete %s: %v", f, err)
		}
	}
}
