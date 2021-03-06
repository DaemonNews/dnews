VERSION=`git describe --tags 2>/dev/null || git log -n 1 --format="%h"`

all: glide get-deps

glide:
	go get github.com/Masterminds/glide

get-deps: glide
	glide i

db:
	psql < sql/postgres.sql

test: db
	sh test/add_articles

build: glide
	go vet
	go build -ldflags "-X main.version=${VERSION}" github.com/DaemonNews/dnews
	go build -ldflags "-X main.version=${VERSION}" github.com/DaemonNews/dnews/cmd/...
