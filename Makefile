
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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

test:
	go test -v ./...
