package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/brooke-hamilton/git-infra-graph/src/graph"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	jsonMode := hasFlag("--json")

	switch command {
	case "init":
		runInit(jsonMode)
	case "list":
		runList(jsonMode)
	case "delete":
		runDelete(jsonMode)
	default:
		printError(jsonMode, fmt.Sprintf("unknown command: %s", command))
		os.Exit(1)
	}
}

func runInit(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "init requires a graph name")
		os.Exit(1)
	}
	name := args[0]
	repoPath := "."

	if err := graph.Init(repoPath, name); err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		// Get the source commit for JSON output
		info, err := graph.GetInitInfo(repoPath, name)
		if err != nil {
			// Init succeeded but info retrieval failed; output minimal JSON
			out, _ := json.Marshal(map[string]string{
				"name": name,
				"ref":  "refs/infra/" + name,
			})
			fmt.Println(string(out))
		} else {
			out, _ := json.Marshal(info)
			fmt.Println(string(out))
		}
	} else {
		fmt.Printf("Initialized graph %q\n", name)
	}
}

func runList(jsonMode bool) {
	repoPath := "."

	graphs, err := graph.List(repoPath)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		names := make([]string, len(graphs))
		for i, g := range graphs {
			names[i] = g.Name
		}
		out, _ := json.Marshal(names)
		fmt.Println(string(out))
	} else {
		for _, g := range graphs {
			fmt.Println(g.Name)
		}
	}
}

func runDelete(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "delete requires a graph name")
		os.Exit(1)
	}
	name := args[0]
	repoPath := "."

	if err := graph.Delete(repoPath, name); err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(map[string]interface{}{
			"name":    name,
			"deleted": true,
		})
		fmt.Println(string(out))
	} else {
		fmt.Printf("Deleted graph %q\n", name)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif <command> [--json] [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init <name>    Create a new infrastructure graph")
	fmt.Fprintln(os.Stderr, "  list           List all infrastructure graphs")
	fmt.Fprintln(os.Stderr, "  delete <name>  Delete an infrastructure graph")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json         Output in JSON format")
}

func printError(jsonMode bool, msg string) {
	if jsonMode {
		out, _ := json.Marshal(map[string]string{"error": msg})
		fmt.Fprintln(os.Stderr, string(out))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

// hasFlag checks if a flag is present in os.Args.
func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

// positionalArgs returns non-flag arguments after the command name.
func positionalArgs() []string {
	var args []string
	for _, arg := range os.Args[2:] {
		if arg == "--json" {
			continue
		}
		args = append(args, arg)
	}
	return args
}
