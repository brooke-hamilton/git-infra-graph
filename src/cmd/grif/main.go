package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

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
	case "put":
		runPut(jsonMode)
	case "get":
		runGet(jsonMode)
	case "rm":
		runRm(jsonMode)
	case "commit":
		runCommit(jsonMode)
	case "status":
		runStatus(jsonMode)
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

func runPut(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "put requires a path")
		os.Exit(1)
	}
	nodePath := args[0]
	repoPath := "."

	dataPresent := hasFlag("--data")
	filePresent := hasFlag("--file")
	dataVal := flagValue("--data")

	// Determine blob content — mutually exclusive sources
	var blob []byte

	// Detect piped stdin
	stat, _ := os.Stdin.Stat()
	stdinPiped := (stat.Mode() & os.ModeCharDevice) == 0

	// Validate mutual exclusivity of input sources
	sourceCount := 0
	if dataPresent {
		sourceCount++
	}
	if filePresent {
		sourceCount++
	}
	if stdinPiped {
		sourceCount++
	}
	if sourceCount > 1 {
		printError(jsonMode, "--data, --file, and stdin are mutually exclusive")
		os.Exit(1)
	}

	switch {
	case dataPresent:
		blob = []byte(dataVal)
	case filePresent:
		content, err := os.ReadFile(flagValue("--file"))
		if err != nil {
			printError(jsonMode, err.Error())
			os.Exit(1)
		}
		blob = content
	case stdinPiped:
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			printError(jsonMode, err.Error())
			os.Exit(1)
		}
		blob = content
		// else: nil blob → tree node
	}

	result, err := graph.Put(repoPath, nodePath, blob)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(result)
		fmt.Println(string(out))
	} else {
		shortID := result.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		fmt.Printf("Put %s at %q (id: %s)\n", result.Type, result.Path, shortID)
	}
}

func runGet(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "get requires a path")
		os.Exit(1)
	}
	nodePath := args[0]
	repoPath := "."

	content, err := graph.Get(repoPath, nodePath)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(content)
		fmt.Println(string(out))
	} else if content.Type == graph.BlobNode {
		fmt.Print(string(content.Blob))
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tNAME\tID")
		for _, child := range content.Children {
			shortID := child.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", child.Type, child.Name, shortID)
		}
		w.Flush()
	}
}

func runRm(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "rm requires a path")
		os.Exit(1)
	}
	nodePath := args[0]
	repoPath := "."

	// Get node info and count descendants BEFORE deleting, so recursive
	// Get calls can still reach child nodes in the staging tree.
	content, getErr := graph.Get(repoPath, nodePath)
	var descendantCount int
	if getErr == nil && content.Type == graph.TreeNode {
		descendantCount = countDescendants(repoPath, nodePath, content)
	}

	if err := graph.DeleteNode(repoPath, nodePath); err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		result := map[string]interface{}{
			"path":    nodePath,
			"removed": true,
		}
		if getErr == nil && content.Type == graph.TreeNode {
			result["type"] = "tree"
			result["descendants"] = descendantCount
		}
		out, _ := json.Marshal(result)
		fmt.Println(string(out))
	} else {
		if getErr == nil && content.Type == graph.TreeNode {
			fmt.Printf("Removed %q (tree, %d descendants)\n", nodePath, descendantCount)
		} else {
			fmt.Printf("Removed %q\n", nodePath)
		}
	}
}

// countDescendants recursively counts all descendants of a tree node.
func countDescendants(repoPath string, basePath string, content *graph.NodeContent) int {
	count := 0
	for _, child := range content.Children {
		count++
		if child.Type == graph.TreeNode {
			childPath := basePath + "/" + child.Name
			childContent, err := graph.Get(repoPath, childPath)
			if err == nil {
				count += countDescendants(repoPath, childPath, childContent)
			}
		}
	}
	return count
}

func runCommit(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "commit requires a graph name")
		os.Exit(1)
	}
	graphName := args[0]
	repoPath := "."

	message := flagValue("--message")

	result, err := graph.Commit(repoPath, graphName, message)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(result)
		fmt.Println(string(out))
	} else {
		shortHash := result.Commit
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}
		fmt.Printf("Committed graph %q (commit: %s)\n", graphName, shortHash)
	}
}

func runStatus(jsonMode bool) {
	args := positionalArgs()
	if len(args) < 1 {
		printError(jsonMode, "status requires a graph name")
		os.Exit(1)
	}
	graphName := args[0]
	repoPath := "."

	result, err := graph.Status(repoPath, graphName)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(result)
		fmt.Println(string(out))
	} else if len(result.Changes) == 0 {
		fmt.Printf("No uncommitted changes for graph %q\n", graphName)
	} else {
		fmt.Printf("Changes for graph %q:\n", graphName)
		for _, change := range result.Changes {
			fmt.Printf("  %-10s %s\n", change.Status+":", change.Path)
		}
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif <command> [--json] [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init <name>       Create a new infrastructure graph")
	fmt.Fprintln(os.Stderr, "  list              List all infrastructure graphs")
	fmt.Fprintln(os.Stderr, "  delete <name>     Delete an infrastructure graph")
	fmt.Fprintln(os.Stderr, "  put <path>        Create or replace a node")
	fmt.Fprintln(os.Stderr, "  get <path>        Read a node")
	fmt.Fprintln(os.Stderr, "  rm <path>         Delete a node")
	fmt.Fprintln(os.Stderr, "  commit <graph>    Commit staged changes")
	fmt.Fprintln(os.Stderr, "  status <graph>    Show uncommitted changes")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  --data <content>  Blob content as string (put)")
	fmt.Fprintln(os.Stderr, "  --file <file>     Read blob content from file (put)")
	fmt.Fprintln(os.Stderr, "  --message <msg>   Commit message (commit)")
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
	skip := false
	for _, arg := range os.Args[2:] {
		if skip {
			skip = false
			continue
		}
		if arg == "--json" {
			continue
		}
		if arg == "--data" || arg == "--file" || arg == "--message" {
			skip = true
			continue
		}
		args = append(args, arg)
	}
	return args
}

// flagValue returns the value of a flag like --data <value>, or "" if not present.
func flagValue(flag string) string {
	allArgs := os.Args[2:]
	for i, arg := range allArgs {
		if arg == flag && i+1 < len(allArgs) {
			return allArgs[i+1]
		}
	}
	return ""
}
