# Go

`go build ./...` build all packages ^run
`go build -o {{name}} .` build with output name ^run:name
`go run .` run current package ^run
`go test ./...` run all tests ^run
`go test -v ./{{pkg}}` run package tests verbose ^run:pkg
`go test -run {{pattern}} ./...` run matching tests ^run:pattern
`go test -cover ./...` run tests with coverage ^run
`go test -bench=. ./...` run benchmarks ^run
`go fmt ./...` format all files ^run
`go vet ./...` static analysis ^run
`go mod init {{module}}` init module ^run:module
`go mod tidy` clean up dependencies ^run
`go mod download` download dependencies ^run
`go mod vendor` vendor dependencies ^run
`go get {{pkg}}` add dependency ^run:pkg
`go get -u ./...` update all dependencies ^run
`go install {{pkg}}` install binary ^run:pkg
`go list ./...` list packages ^run
`go list -m all` list all modules ^run
`go doc {{symbol}}` show documentation ^run:symbol
`go env` show environment ^run
`go generate ./...` run go generate ^run
`go clean -cache` clean build cache ^run
`go clean -testcache` clean test cache ^run
`go tool pprof {{file}}` profile analysis ^run:file
`golangci-lint run ./...` run linter ^run
