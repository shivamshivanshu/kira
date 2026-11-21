set positional-arguments

# List the available recipes.
default:
    @just --list

# Build the local CLI with symbols in ./kira.
build:
    go build -o ./kira ./cmd/kira

# Build a stripped, non-CGO CLI in ./kira using the release flags.
release version="dev":
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/shivamshivanshu/kira/internal/cli.version=$1" -o ./kira ./cmd/kira

# Install the release binary in /usr/local/bin or a directory you specify.
install bindir="/usr/local/bin": release
    mkdir -p "$1"
    install -m 0755 ./kira "$1/"

# Build ./kira-debug with optimization and inlining disabled.
debug:
    go build -gcflags='all=-N -l' -o ./kira-debug ./cmd/kira

# Remove local binaries and the repository's rebuildable SQLite index.
clean:
    rm -f ./kira ./kira-debug .kira/.cache/index.db .kira/.cache/index.db-wal .kira/.cache/index.db-shm .kira/.cache/meta.json

# Run the CLI from source.
run *args:
    go run ./cmd/kira "$@"

# Run tests. Pass packages and flags to narrow the run.
test +args="./...":
    env -u NO_COLOR go test "$@"

# Run tests with the race detector.
test-race +args="./...":
    env -u NO_COLOR go test -race "$@"

# Run Go's static checks.
vet:
    go vet ./...

# Run golangci-lint using the version pinned by CI.
lint:
    golangci-lint run ./...

# Run the build, vet, race, and lint checks used by CI.
check: vet test-race lint
    go build ./...

# Run the performance checks used by CI.
perf:
    KIRA_PERF=1 env -u NO_COLOR go test -count=1 -v -run 'TestSpawnCounts|TestScaling|TestColdStartSmoke|TestColdStatsSpawnScaling' ./tests/perf
