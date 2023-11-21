package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RepoCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "repo_found_total_count",
		Help: "The total number of repos found in organization",
	}, []string{"org"})
	BackupTotalCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "repo_backup_total_count",
		Help: "The total number of repos backed up",
	}, []string{"org", "medium"})
	BackupFailureCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "repo_backup_failure_count",
		Help: "The total number of repos that failed backup",
	}, []string{"org", "medium"})
	BackupSuccessCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "repo_backup_success_count",
		Help: "The total number of repos that backed up OK",
	}, []string{"org", "medium"})
)
