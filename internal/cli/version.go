package cli

// version is the build version, overridable at link time:
//
//	go build -ldflags "-X github.com/shivamshivanshu/kira/internal/cli.version=v1.2.3"
var version = "dev"
