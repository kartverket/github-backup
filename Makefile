all: github-backup


github-backup:
	go build -o bin/github-backup cmd/github-backup/*.go
