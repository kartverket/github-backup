# github-backup

Backup all repos in ORG_NAMES env as zip files to a GCS bucket or NFS share.

## Usage

This service is run as a Job at set intervals. Needs the following environment variables set.

| Env          | Value                                                 | Required                  |
|--------------|-------------------------------------------------------|---------------------------|
| ORG_NAMES    | Comma seperated list of github orgs. "org1,org2,org3" | Yes                       |
| NFS_SHARE    | Share to use for backup of repos                      | Yes, if BUCKET_NAME is NA |
| BUCKET_NAME  | Name of GCS bucket                                    | Yes, if NFS_SHARE is NA   |
| GITHUB_TOKEN | Token for connecting to Github                        | Yes                       | 
| GITHUB_USER | Username to connect to Github | Yes |        
