all: github-backup

test:
	go test ./... -coverprofile cover.out

github-backup:
	go build -o bin/github-backup cmd/github-backup/*.go

