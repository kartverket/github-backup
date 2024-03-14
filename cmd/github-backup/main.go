package main

import (
	"context"
	"github-backup/pkg/git"
	"github-backup/pkg/metrics"
	"github-backup/pkg/nfs-cleanup"
	"github-backup/pkg/objstorage"
	"github-backup/pkg/zippings"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var basedir = filepath.Join(os.TempDir(), "ghbackup")

const MaxConcurrent = 10

func main() {
	//metrics

	http.Handle(
		"/metrics",
		promhttp.Handler(),
	)
	go func() {
		err := http.ListenAndServe(":8080", nil)
		log.Fatal().Msgf("%v", err)
	}()

	var bucketname string
	var goog *storage.Client
	var timeToLive string

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
			err := os.RemoveAll(path)
			if err != nil {
				log.Error().Msgf("Error during cleanup: %v", err)
			}
		}
		log.Info().Msgf("Using NFS storage on with directory '%s'", nfsShare)
	}
	if nfsStorage {
		timeToLive = envOrDie("TIME_TO_LIVE")
		fileList := []string{}
		//Cleanup old files
		files, err := nfscleanup.ListFiles(nfsShare)
		if err != nil {
			log.Error().Msgf("Error listing files: %v", err)
		}
		ttl, err := strconv.ParseFloat(timeToLive, 64)
			if err != nil {
				log.Error().Msgf("Error parsing time to live (%v) this needs to be parsable as a float64 value", err)
			}
		log.Info().Msgf("Time-To-Live for files set to: %vh", ttl)
		for _, file := range files {
			fileAge, err := nfscleanup.FindFileAge(file)
			if err != nil {
				log.Error().Msgf("Error finding file age: %v", err)
			}
		
			if fileAge.Hours() > ttl {
				err := os.Remove(file)
				if err != nil {
					log.Error().Msgf("Error removing file: %v", err)
				} else {
					log.Info().Msgf("File %s removed", file)
				}
			} else {
				fileList = append(fileList, file)
			}
		}
		log.Info().Msgf("Files remaining after cleanup: %v", len(fileList))
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
			var err error
			var medium = "undefined"

			if !nfsStorage {
				medium = "gcs"
				err = cloneZipAndStoreInBucket(r.FullName, bucketname, userName, githubToken, goog)
				if err != nil {
					log.Error().Msgf("failed to backup repo '%s': %v", r.FullName, err)
				}
			} else {
				medium = "nfs"
				err = cloneZipAndStoreInFile(r.FullName, nfsShare, userName, githubToken)
				if err != nil {
					log.Error().Msgf("failed to backup repo '%s': %v", r.FullName, err)
				}
			}

			if err != nil {
				metrics.BackupFailureCount.WithLabelValues(r.Owner.Login, medium).Inc()
			} else {
				metrics.BackupSuccessCount.WithLabelValues(r.Owner.Login, medium).Inc()
			}
			metrics.BackupTotalCount.WithLabelValues(r.Owner.Login, medium).Inc()

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
