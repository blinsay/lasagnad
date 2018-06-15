NAME := lasagnad
DBNAME := garf.db
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
	go build -o $(NAME) .

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
.PHONY: clean
clean:
	@echo "+ $@"
	$(RM) $(NAME)

# create a new DB from scratch
# .PHONY: db-init
# db-init:
# 	@echo "+ $@"
# 	sqlite3 $(DBNAME) < schema.sql

# .PHONY: db-reset
# db-reset:
# 	@echo "+ $@"
# 	rm -f garf.db
# 	sqlite3 $(DBNAME) < schema.sql

# # dump the DB schema to a SQL file
# .PHONY: db-dump-schema
# db-dump-schema:
# 	@echo "+ $@"
# 	sqlite3 $(DBNAME) .schema > schema.sql
