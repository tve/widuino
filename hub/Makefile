# Makefile for simple Golang projects
NAME=hub

build: $(NAME)
$(NAME): *.go */*.go
	go build -o $(NAME)

verbosetest: $(NAME)
	ginkgo -noColor -- -logtostderr

test: $(NAME)
	ginkgo -noColor
	ginkgo -noColor -cover
	go tool cover -func=$(NAME).coverprofile
