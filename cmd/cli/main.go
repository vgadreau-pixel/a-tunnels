package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/a-tunnels/a-tunnels/internal/i18n"
	"github.com/a-tunnels/a-tunnels/pkg/sdk/go"
	"github.com/fatih/color"
)

func init() {
	_ = color.NoColor
}

var (
	serverURL  string
	token      string
	outputJSON bool
	noColor    bool
	lang       string
)

var (
	cyan    = color.New(color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
	dim     = color.New(color.Faint).SprintFunc()
)

func main() {
	flag.StringVar(&serverURL, "server", "http://localhost:8080", i18n.T("options.server"))
	flag.StringVar(&token, "token", "", i18n.T("options.token"))
	flag.BoolVar(&outputJSON, "json", false, i18n.T("options.json"))
	flag.BoolVar(&noColor, "no-color", false, i18n.T("options.no_color"))
	flag.StringVar(&lang, "lang", "", i18n.T("options.lang"))
	flag.Usage = usage
	flag.Parse()

	i18n.Init()
	if lang != "" {
		i18n.SetLang(lang)
	}

	if noColor {
		color.NoColor = true
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if token == "" {
		token = os.Getenv("ATUNNELS_TOKEN")
	}

	if token == "" {
		fmt.Fprintln(os.Stderr, red("Error:"), i18n.TError("token_required"))
		os.Exit(1)
	}

	client := atunnels.NewClient(serverURL, token)

	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	var err error
	switch cmd {
	case "help", "h", "-h", "--help":
		usage()
		os.Exit(0)
	case "list", "ls":
		err = listTunnels(client)
	case "get":
		err = getTunnel(client, args)
	case "create":
		err = createTunnel(client, args)
	case "delete", "rm":
		err = deleteTunnel(client, args)
	case "stats":
		err = getStats(client, args)
	case "restart":
		err = restartTunnel(client, args)
	case "logs":
		err = getLogs(client, args)
	case "health":
		err = checkHealth(client)
	case "version", "v":
		showVersion()
	default:
		fmt.Fprintf(os.Stderr, red(i18n.TError("unknown_command")+":"), " %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error:"), err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, bold(cyan("╔═══════════════════════════════════════════╗")))
	fmt.Fprintln(os.Stderr, bold(cyan("║")), bold("  "+i18n.T("app.name")+"                    "), bold(cyan("║")))
	fmt.Fprintln(os.Stderr, bold(cyan("╚═══════════════════════════════════════════╝")))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow(i18n.T("usage.title"))))
	fmt.Fprintln(os.Stderr, bold("  a-tunnels [options] <command> [args]"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow("Commands:")))
	fmt.Fprintln(os.Stderr, bold("  list, ls            "), dim(i18n.TCommand("list")))
	fmt.Fprintln(os.Stderr, bold("  get <name>          "), dim(i18n.TCommand("get")))
	fmt.Fprintln(os.Stderr, bold("  create [name] <proto> <addr>"), dim(i18n.TCommand("create")))
	fmt.Fprintln(os.Stderr, bold("  delete, rm <name>  "), dim(i18n.TCommand("delete")))
	fmt.Fprintln(os.Stderr, bold("  stats <name>        "), dim(i18n.TCommand("stats")))
	fmt.Fprintln(os.Stderr, bold("  restart <name>      "), dim(i18n.TCommand("restart")))
	fmt.Fprintln(os.Stderr, bold("  logs <name>         "), dim(i18n.TCommand("logs")))
	fmt.Fprintln(os.Stderr, bold("  health              "), dim(i18n.TCommand("health")))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow(i18n.T("options.server")+":")))
	fmt.Fprintln(os.Stderr, bold("  -server URL         "), dim(i18n.T("options.server")+" (default: http://localhost:8080)"))
	fmt.Fprintln(os.Stderr, bold("  -token string       "), dim(i18n.T("options.token")))
	fmt.Fprintln(os.Stderr, bold("  -json               "), dim(i18n.T("options.json")))
	fmt.Fprintln(os.Stderr, bold("  -no-color           "), dim(i18n.T("options.no_color")))
	fmt.Fprintln(os.Stderr, bold("  -lang <en|fr|es>    "), dim(i18n.T("options.lang")))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow(i18n.T("usage.example"))))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx list"))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx create webhook http localhost:3000"))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx stats webhook"))
}

func showVersion() {
	fmt.Printf("%s %s v%s\n", bold(i18n.T("app.name")), bold(i18n.T("app.version")), green("1.0.0"))
	fmt.Printf("%s: %s\n", dim("Build"), "go")
}

func listTunnels(client *atunnels.Client) error {
	tunnels, err := client.ListTunnels()
	if err != nil {
		return err
	}

	if outputJSON {
		json.NewEncoder(os.Stdout).Encode(tunnels)
		return nil
	}

	if len(tunnels) == 0 {
		fmt.Println(dim(i18n.T("messages.no_tunnels")))
		return nil
	}

	fmt.Println()
	fmt.Printf("%s  %s  %s  %s  %s\n",
		bold(i18n.T("tunnel.id")), bold(i18n.T("tunnel.name")), bold(i18n.T("tunnel.protocol")), bold(i18n.T("tunnel.local_addr")), bold(i18n.T("tunnel.status")))
	fmt.Println(dim(strings.Repeat("─", 70)))
	for _, t := range tunnels {
		statusColor := yellow
		if t.Status == "active" {
			statusColor = green
		} else if t.Status == "error" {
			statusColor = red
		}
		fmt.Printf("%s  %s  %s  %s  %s\n",
			dim(t.ID[:min(8, len(t.ID))]),
			cyan(t.Name),
			magenta(t.Protocol),
			dim(t.LocalAddr),
			statusColor(i18n.TStatus(t.Status)))
	}
	fmt.Println()
	return nil
}

func getTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(i18n.TError("get_usage"))
	}

	name := args[0]
	tunnel, err := client.GetTunnel(name)
	if err != nil {
		return err
	}

	if outputJSON {
		json.NewEncoder(os.Stdout).Encode(tunnel)
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.name")+":"), cyan(tunnel.Name))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.id")+":"), dim(tunnel.ID))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.protocol")+":"), magenta(tunnel.Protocol))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.local_addr")+":"), dim(tunnel.LocalAddr))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.status")+":"), green(i18n.TStatus(tunnel.Status)))
	if tunnel.Subdomain != "" {
		fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.subdomain")+":"), cyan(tunnel.Subdomain))
	}
	fmt.Println()
	return nil
}

func createTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(i18n.TError("create_usage"))
	}

	var name, protocol, localAddr string

	if len(args) == 2 {
		generatedName, err := client.GenerateName()
		if err != nil {
			return fmt.Errorf(i18n.TError("failed_to_generate_name")+": %w", err)
		}
		name = generatedName
		protocol = args[0]
		localAddr = args[1]
	} else {
		name = args[0]
		protocol = args[1]
		localAddr = args[2]
	}

	tunnel, err := client.CreateTunnel(&atunnels.Tunnel{
		Name:      name,
		Protocol:  protocol,
		LocalAddr: localAddr,
	})
	if err != nil {
		return err
	}

	if outputJSON {
		json.NewEncoder(os.Stdout).Encode(tunnel)
		return nil
	}

	fmt.Printf("%s %s (%s)\n", green("✓"), bold(i18n.T("tunnel.created")), cyan(tunnel.Name))
	fmt.Printf("  %s: %s\n", dim(i18n.T("tunnel.id")), tunnel.ID)
	return nil
}

func deleteTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(i18n.TError("delete_usage"))
	}

	name := args[0]
	err := client.DeleteTunnel(name)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", green("✓"), bold(i18n.T("tunnel.deleted")))
	return nil
}

func getStats(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(i18n.TError("stats_usage"))
	}

	name := args[0]
	stats, err := client.GetTunnelStats(name)
	if err != nil {
		return err
	}

	if outputJSON {
		json.NewEncoder(os.Stdout).Encode(stats)
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.active_connections")+":"), cyan(stats.ActiveConnections))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.total_requests")+":"), cyan(stats.TotalRequests))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.total_bytes_in")+":"), green(stats.TotalBytesIn))
	fmt.Printf("  %s  %s\n", bold(i18n.T("tunnel.total_bytes_out")+":"), magenta(stats.TotalBytesOut))
	fmt.Println()
	return nil
}

func restartTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: restart <name>")
	}

	name := args[0]
	err := client.RestartTunnel(name)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", green("✓"), bold(i18n.T("tunnel.restarted")))
	return nil
}

func getLogs(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: logs <name>")
	}

	name := args[0]

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/tunnels/%s/logs", serverURL, name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Println(string(data))
	return nil
}

func checkHealth(client *atunnels.Client) error {
	healthy, err := client.Health()
	if err != nil {
		return err
	}

	if outputJSON {
		json.NewEncoder(os.Stdout).Encode(map[string]bool{"healthy": healthy})
		return nil
	}

	if healthy {
		fmt.Printf("%s %s\n", green("✓"), bold(i18n.T("messages.server_healthy")))
	} else {
		fmt.Printf("%s %s\n", red("✗"), bold(i18n.T("messages.server_unhealthy")))
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
