# Makefile for nodegw
NAME=nodegw

build: $(NAME)
$(NAME): *.go
	go build -o $(NAME)

test: $(NAME)
	ginkgo -noColor
	ginkgo -noColor -cover
	go tool cover -func=$(NAME).coverprofile
