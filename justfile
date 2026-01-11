# Build and install the extension locally
install:
    go build -o gh-subissue
    -gh extension remove subissue
    gh extension install .

# Run all tests
test:
    go test ./...

# Remove the extension from gh
remove:
    gh extension remove subissue

alias delete := remove

# Check if issues are enabled for a repo (defaults to current repo)
check-issues repo="":
    gh repo view {{repo}} --json hasIssuesEnabled
