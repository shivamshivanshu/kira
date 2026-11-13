package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/webui"
)

func newServerCmd(g *globalFlags) *cobra.Command {
	var listen string
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"serve"},
		Short:   "Serve the local web dashboard for this kira store",
		Long: "Serve a local dashboard with tree, board, stats, and item editing.\n" +
			"Open the printed URL in your browser; Ctrl+C stops the server.\n" +
			"Actions use the same workflows and commit policy as the CLI.\n" +
			"Only loopback addresses are supported.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if g.json {
				return errx.User("server: --json is not supported for the web dashboard")
			}
			if g.nonInteractive {
				return errx.User("cannot launch the server in non-interactive mode")
			}
			addr, err := serverAddress(listen)
			if err != nil {
				return err
			}
			s, _, err := openStore(g)
			if err != nil {
				return err
			}
			handler, err := webui.New(s.WithPrompter(core.SilentPrompter()), webui.Options{RunCommand: serverCommandRunner(s.Root())})
			if err != nil {
				return err
			}
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return errx.Env("server: %v", err).WithHint("choose another port with --listen 127.0.0.1:7433")
			}
			defer func() { _ = ln.Close() }()
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "kira dashboard: http://%s\n", ln.Addr()); err != nil {
				return err
			}
			return serveDashboard(ctx, ln, handler)
		},
	}
	cmd.Flags().StringVar(&listen, "listen", "127.0.0.1:7432", "loopback host:port to listen on (port 0 chooses a free port)")
	return cmd
}

func serverAddress(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", errx.User("server: invalid listen address %q: %v", addr, err)
	}
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", errx.User("server: listen address must use localhost or a loopback IP")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return "", errx.User("server: invalid port %q", port)
	}
	return net.JoinHostPort(ip.String(), strconv.Itoa(n)), nil
}

func serveDashboard(ctx context.Context, ln net.Listener, handler http.Handler) error {
	// Leave writes unbounded so long-running Git or hook actions can report their result.
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return err
		}
		return nil
	}
}

func serverCommandRunner(storeRoot string) func([]string) (string, error) {
	return func(argv []string) (string, error) {
		if len(argv) == 0 || strings.HasPrefix(argv[0], "-") {
			return "", errx.User("enter a kira command, for example: list")
		}
		var buf bytes.Buffer
		root, g := newRootCmd()
		g.chdir = storeRoot
		g.nonInteractive = true
		g.noColor = true
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetIn(strings.NewReader(""))
		root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
			if root.PersistentFlags().Changed("C") || !g.nonInteractive {
				return errx.User("dashboard commands must stay in the current store and run non-interactively")
			}
			if !dashboardCommandAllowed(cmd.CommandPath()) {
				return errx.User("%s is unavailable in the dashboard; run it in your terminal", cmd.CommandPath())
			}
			if cmd.Flags().Changed("from-file") {
				return errx.User("dashboard commands cannot read --from-file; use the item form")
			}
			switch cmd.Name() {
			case "workon":
				worktree, err := cmd.Flags().GetBool("worktree")
				if err != nil {
					return err
				}
				if worktree {
					return errx.User("dashboard workon stays in the current checkout; create worktrees in your terminal")
				}
				return cmd.Flags().Set("worktree", "false")
			case "sync":
				remote, err := cmd.Flags().GetString("remote")
				if err != nil {
					return err
				}
				return validateDashboardRemote(storeRoot, remote)
			}
			return nil
		}
		root.SetArgs(argv)
		err := root.Execute()
		return buf.String(), err
	}
}

func dashboardCommandAllowed(path string) bool {
	switch strings.TrimPrefix(path, "kira ") {
	case "list", "tree", "board", "show", "stats", "now", "log", "blame", "diff", "changes", "find",
		"create", "edit", "move", "assign", "link", "comment", "commit", "workon", "sync", "doctor", "index", "version", "schema",
		"board list", "board create", "board rename", "board archive", "board unarchive", "board move",
		"sprint list", "sprint create", "sprint activate", "sprint close",
		"label list", "label create", "label add", "label rm", "config explain", "config filters", "config set",
		"hooks status":
		return true
	default:
		return false
	}
}

func validateDashboardRemote(storeRoot, remote string) error {
	if remote == "" {
		return nil
	}
	if strings.HasPrefix(remote, "-") {
		return errx.User("dashboard sync requires a configured remote name")
	}
	remotes, err := (gitx.Repo{Dir: storeRoot}).Output("remote")
	if err != nil {
		return errx.User("listing remotes: %v", err)
	}
	if !slices.Contains(strings.Split(remotes, "\n"), remote) {
		return errx.User("dashboard sync requires a configured remote name; configure remote URLs in your terminal")
	}
	return nil
}
