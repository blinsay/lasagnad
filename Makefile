NAME := lasagnad
PKG := github.com/blinsay/lasagnad

all: clean deps fmt lint vet

.PHONY: deps
deps:
	@echo "+ $@"
	@dep ensure

# build a binary
.PHONY: build
build: $(NAME)

.PHONY: $(NAME)
$(NAME): *.go
	@echo "+ $@"
	go build -o $(NAME) ./cmd/lasagnad

# make sure gofmt was run
.PHONY: fmt
fmt:
	@echo "+ $@"
	@gofmt -s -l . 2>&1 | grep -v .pb.go | grep -v vendor | tee /dev/stderr

# golint
.PHONY: lint
lint:
	@echo "+ $@"
	@golint ./... 2>&1 | grep -v .pb.go | grep -v vendor | tee /dev/stderr

# go vet
.PHONY: vet
vet:
	@echo "+ $@"
	@go vet ./...

# clean up local executeables
.PHONY: .clean
clean:
	@echo "+ $@"
	$(RM) $(NAME)
