FROM golang:alpine

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o ./bin/github-backup ./cmd/github-backup/main.go

USER 150:150

CMD ["sh","-c","sleep 500 && ./bin/github-backup"]
