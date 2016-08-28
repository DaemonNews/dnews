all: glide get-deps

glide:
	go get github.com/Masterminds/glide

get-deps: glide
	glide i

build: glide
	go vet
	go build
