
sanity: goimport
	go fmt ./...
	go mod tidy -v
	go mod vendor
	git add -N vendor
	git diff --exit-code

goimport:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w -local="github.com/machadovilaca/alerts-ui-management" .

lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.1
	golangci-lint run ./...

test:
	go test -v ./...

setup-hooks:
	cp hack/git/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks setup successfully!"
