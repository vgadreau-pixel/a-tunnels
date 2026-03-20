package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL")
	flag.StringVar(&token, "token", "", "API token")
	flag.BoolVar(&outputJSON, "json", false, "Output JSON")
	flag.BoolVar(&noColor, "no-color", false, "Disable colors")
	flag.Usage = usage
	flag.Parse()

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
		fmt.Fprintln(os.Stderr, red("Error:"), "token required (use -token or ATUNNELS_TOKEN)")
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
		fmt.Fprintf(os.Stderr, red("Unknown command:"), " %s\n", cmd)
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
	fmt.Fprintln(os.Stderr, bold(cyan("║")), bold("  A-Tunnels CLI                    "), bold(cyan("║")))
	fmt.Fprintln(os.Stderr, bold(cyan("╚═══════════════════════════════════════════╝")))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow("Usage:")))
	fmt.Fprintln(os.Stderr, bold("  a-tunnels [options] <command> [args]"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow("Commands:")))
	fmt.Fprintln(os.Stderr, bold("  list, ls            "), dim("List all tunnels"))
	fmt.Fprintln(os.Stderr, bold("  get <name>          "), dim("Get tunnel details"))
	fmt.Fprintln(os.Stderr, bold("  create [name] <proto> <addr>"), dim("Create tunnel (name is optional)"))
	fmt.Fprintln(os.Stderr, bold("  delete, rm <name>  "), dim("Delete tunnel"))
	fmt.Fprintln(os.Stderr, bold("  stats <name>        "), dim("Get tunnel statistics"))
	fmt.Fprintln(os.Stderr, bold("  restart <name>      "), dim("Restart tunnel"))
	fmt.Fprintln(os.Stderr, bold("  logs <name>         "), dim("Get tunnel logs"))
	fmt.Fprintln(os.Stderr, bold("  health              "), dim("Check server health"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow("Options:")))
	fmt.Fprintln(os.Stderr, bold("  -server URL         "), dim("Server URL (default: http://localhost:8080)"))
	fmt.Fprintln(os.Stderr, bold("  -token string       "), dim("API token"))
	fmt.Fprintln(os.Stderr, bold("  -json               "), dim("Output JSON"))
	fmt.Fprintln(os.Stderr, bold("  -no-color           "), dim("Disable colors"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold(yellow("Examples:")))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx list"))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx create webhook http localhost:3000"))
	fmt.Fprintln(os.Stderr, dim("  a-tunnels -token xxx stats webhook"))
}

func showVersion() {
	fmt.Printf("%s v%s\n", bold("A-Tunnels CLI"), green("1.0.0"))
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
		fmt.Println(dim("No tunnels"))
		return nil
	}

	fmt.Println()
	fmt.Printf("%s  %s  %s  %s  %s\n",
		bold("ID"), bold("NAME"), bold("PROTOCOL"), bold("LOCAL_ADDR"), bold("STATUS"))
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
			statusColor(t.Status))
	}
	fmt.Println()
	return nil
}

func getTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get <name>")
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
	fmt.Printf("  %s  %s\n", bold("Name:"), cyan(tunnel.Name))
	fmt.Printf("  %s  %s\n", bold("ID:"), dim(tunnel.ID))
	fmt.Printf("  %s  %s\n", bold("Protocol:"), magenta(tunnel.Protocol))
	fmt.Printf("  %s  %s\n", bold("LocalAddr:"), dim(tunnel.LocalAddr))
	fmt.Printf("  %s  %s\n", bold("Status:"), green(tunnel.Status))
	if tunnel.Subdomain != "" {
		fmt.Printf("  %s  %s\n", bold("Subdomain:"), cyan(tunnel.Subdomain))
	}
	fmt.Println()
	return nil
}

func createTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: create [name] <protocol> <localAddr>")
	}

	var name, protocol, localAddr string

	if len(args) == 2 {
		generatedName, err := client.GenerateName()
		if err != nil {
			return fmt.Errorf("failed to generate name: %w", err)
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

	fmt.Printf("%s %s (%s)\n", green("✓"), bold("Tunnel created"), cyan(tunnel.Name))
	fmt.Printf("  %s: %s\n", dim("ID"), tunnel.ID)
	return nil
}

func deleteTunnel(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: delete <name>")
	}

	name := args[0]
	err := client.DeleteTunnel(name)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", green("✓"), bold("Tunnel deleted"))
	return nil
}

func getStats(client *atunnels.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: stats <name>")
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
	fmt.Printf("  %s  %s\n", bold("Active Connections:"), cyan(stats.ActiveConnections))
	fmt.Printf("  %s  %s\n", bold("Total Requests:"), cyan(stats.TotalRequests))
	fmt.Printf("  %s  %s\n", bold("Total Bytes In:"), green(stats.TotalBytesIn))
	fmt.Printf("  %s  %s\n", bold("Total Bytes Out:"), magenta(stats.TotalBytesOut))
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

	fmt.Printf("%s %s\n", green("✓"), bold("Tunnel restarted"))
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
		fmt.Printf("%s %s\n", green("✓"), bold("Server is healthy"))
	} else {
		fmt.Printf("%s %s\n", red("✗"), bold("Server is unhealthy"))
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
