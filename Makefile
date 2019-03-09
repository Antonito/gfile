RM			?= rm -f
ECHO		?= echo
GO			?= go

NAME		:= gfile

PKG_LIST	:= $(go list ./... | grep -v /vendor/)

deps:
	@$(ECHO) "==> Installing deps ..."
	@go get ./...

build: deps
	@$(ECHO) "==> Building ..."
	@go build -race -o $(NAME) .

build-all: deps
	@$(ECHO) "==> Building all binaries..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/$(NAME)-macos main.go
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o build/$(NAME)-linux-x86_64 main.go
	@CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -ldflags '-w -extldflags "-static"' -o build/$(NAME)-linux-i386 main.go


lint:
	@$(GOPATH)/bin/golint -set_exit_status ./... | grep -v vendor/ && exit 1 || exit 0

test:
	@$(ECHO) "==> Running tests..."
	@go test -short ${PKG_LIST}

clean:
	@$(RM) $(NAME)

race: deps
	@go test -race -short ${PKG_LIST}

msan: deps
	@go test -msan -short ${PKG_LIST}

coverage:
	@mkdir -p cover/
	@go test ${PKG_LIST} -v -coverprofile cover/testCoverage.txt

coverhtml: coverage
	@go tool cover -html=cover/testCoverage.txt -o cover/coverage.html

.PHONY: deps build build-all test clean lint