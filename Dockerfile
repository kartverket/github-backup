FROM cgr.dev/chainguard/go:latest as build

WORKDIR /go/github-backup
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 make

FROM cgr.dev/chainguard/static:latest
COPY --from=build /go/github-backup/bin /
CMD ["/github-backup"]
