# github-backup

Backup all repositories from specified GitHub organizations as zip files to a Google Cloud Storage (GCS) bucket or an NFS share.

## Features

- Backs up all repositories in one or more GitHub organizations.
- Supports GCS and NFS storage backends.
- Automatic cleanup of old backups (NFS only).
- Exports Prometheus metrics on port `8080`.
- Shallow clones (`--depth 1`) to save space and time.

## Usage

This service is designed to run as a scheduled job. It requires the following environment variables.

### Environment Variables

| Env | Description | Required |
|-----|-------------|----------|
| `ORG_NAMES` | Comma-separated list of GitHub organizations (e.g., `org1,org2`). | Yes |
| `GITHUB_USER` | GitHub username used for authentication. | Yes |
| `GITHUB_TOKEN` | GitHub Personal Access Token (PAT). See [Permissions](#github-token-permissions). | Yes |
| `NFS_SHARE` | Local path to the NFS share. If set, NFS storage is used. | If GCS is not used |
| `BUCKET_NAME` | GCS bucket name. Used if `NFS_SHARE` is not set. | If NFS is not used |
| `TIME_TO_LIVE` | Retention period for backup files in hours (e.g., `24`). | Yes (NFS only) |

### GitHub Token Permissions

The `GITHUB_TOKEN` requires the following scopes:

- **`read:org`**: To list all repositories within the specified organizations.
- **`repo`**: To clone private repositories. If you only need to back up public repositories, this may not be required depending on your organization's settings.

## Storage Backends

### NFS Storage

If `NFS_SHARE` is provided, the tool will:
1.  Store backups as: `${NFS_SHARE}/ghbackup_${org}_${repo}_${timestamp}.zip`.
2.  Clean up any files in `${NFS_SHARE}` older than `TIME_TO_LIVE` hours before starting the backup.

### GCS Storage

If `NFS_SHARE` is not provided, `BUCKET_NAME` must be set. The tool will:
1.  Store backups in the bucket with the path: `YYYY/MM/DD/ghbackup_${org}_${repo}_${timestamp}.zip`.
2.  Requires valid Google Cloud credentials to be configured in the environment (e.g., `GOOGLE_APPLICATION_CREDENTIALS`).

## Development

### Building

To build the binary:

```bash
make github-backup
```

The binary will be located at `bin/github-backup`.

### Running Locally

```bash
export GITHUB_USER=your-user
export GITHUB_TOKEN=your-token
export ORG_NAMES=your-org
export BUCKET_NAME=your-bucket
# or export NFS_SHARE=/path/to/backup && export TIME_TO_LIVE=24

./bin/github-backup
```

### Metrics

Prometheus metrics are exposed at `http://localhost:8080/metrics`.
- `repo_found_total_count`: Number of repositories found in each organization.
- `repo_backup_success_count`: Number of successful backups.
- `repo_backup_failure_count`: Number of failed backups.
- `repo_backup_total_count`: Total number of backup attempts.
