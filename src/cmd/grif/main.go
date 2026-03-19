package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/brooke-hamilton/git-infra-graph/src/graph"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	if command == "--help" || command == "-h" {
		printUsage()
		return
	}

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
	case "tree":
		runTree(jsonMode)
	default:
		printError(jsonMode, fmt.Sprintf("unknown command: %s", command))
		os.Exit(1)
	}
}

func runInit(jsonMode bool) {
	if hasFlag("--help") || hasFlag("-h") {
		printInitUsage()
		return
	}

	args := positionalArgs()
	if len(args) < 1 {
		printInitUsage()
		os.Exit(1)
	}
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("init expects 1 argument, got %d", len(args)))
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
	if hasFlag("--help") || hasFlag("-h") {
		printListUsage()
		return
	}

	args := positionalArgs()
	if len(args) > 0 {
		printError(jsonMode, fmt.Sprintf("list expects no arguments, got %d", len(args)))
		os.Exit(1)
	}

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
	if hasFlag("--help") || hasFlag("-h") {
		printDeleteUsage()
		return
	}

	args := positionalArgs()
	if len(args) < 1 {
		printDeleteUsage()
		os.Exit(1)
	}
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("delete expects 1 argument, got %d", len(args)))
		os.Exit(1)
	}
	name := args[0]
	repoPath := "."

	if err := graph.Delete(repoPath, name); err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(map[string]any{
			"name":    name,
			"deleted": true,
		})
		fmt.Println(string(out))
	} else {
		fmt.Printf("Deleted graph %q\n", name)
	}
}

func runPut(jsonMode bool) {
	if hasFlag("--help") || hasFlag("-h") {
		printPutUsage()
		return
	}

	args := positionalArgs()
	if len(args) < 1 {
		printPutUsage()
		os.Exit(1)
	}
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("put expects 1 argument, got %d", len(args)))
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
	if hasFlag("--help") || hasFlag("-h") {
		printGetUsage()
		return
	}

	args := positionalArgs()
	if len(args) < 1 {
		printGetUsage()
		os.Exit(1)
	}
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("get expects 1 argument, got %d", len(args)))
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
	if hasFlag("--help") || hasFlag("-h") {
		printRmUsage()
		return
	}

	args := positionalArgs()
	if len(args) < 1 {
		printRmUsage()
		os.Exit(1)
	}
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("rm expects 1 argument, got %d", len(args)))
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
		result := map[string]any{
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
	if hasFlag("--help") || hasFlag("-h") {
		printCommitUsage()
		return
	}

	args := positionalArgs()
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("commit expects at most 1 argument, got %d", len(args)))
		os.Exit(1)
	}
	repoPath := "."

	var graphName string
	if len(args) == 1 {
		graphName = args[0]
	} else {
		graphs, err := graph.List(repoPath)
		if err != nil {
			printError(jsonMode, err.Error())
			os.Exit(1)
		}

		switch len(graphs) {
		case 0:
			printError(jsonMode, "no graphs found")
			os.Exit(1)
		case 1:
			graphName = graphs[0].Name
		default:
			printError(jsonMode, "multiple graphs exist; specify a graph name")
			os.Exit(1)
		}
	}

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
	if hasFlag("--help") || hasFlag("-h") {
		printStatusUsage()
		return
	}

	args := positionalArgs()
	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("status expects at most 1 argument, got %d", len(args)))
		os.Exit(1)
	}
	repoPath := "."

	var graphName string
	if len(args) == 1 {
		graphName = args[0]
	} else {
		graphs, err := graph.List(repoPath)
		if err != nil {
			printError(jsonMode, err.Error())
			os.Exit(1)
		}

		switch len(graphs) {
		case 0:
			printError(jsonMode, "no graphs found")
			os.Exit(1)
		case 1:
			graphName = graphs[0].Name
		default:
			printError(jsonMode, "multiple graphs exist; specify a graph name")
			os.Exit(1)
		}
	}

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

func runTree(jsonMode bool) {
	if hasFlag("--help") || hasFlag("-h") {
		printTreeUsage()
		return
	}

	repoPath := "."
	opts := graph.TreeOptions{}

	// Parse --depth flag
	depthStr := flagValue("--depth")
	if depthStr != "" {
		depth, err := strconv.Atoi(depthStr)
		if err != nil {
			printError(jsonMode, fmt.Sprintf("invalid depth value: %s", depthStr))
			os.Exit(1)
		}
		opts.Depth = depth
		opts.HasDepth = true
	}

	args := positionalArgs()

	if len(args) == 0 {
		// All-graphs mode
		result, err := graph.TreeAll(repoPath, opts)
		if err != nil {
			printError(jsonMode, err.Error())
			os.Exit(1)
		}

		if jsonMode {
			out, _ := json.Marshal(result)
			fmt.Println(string(out))
		} else {
			for _, w := range result.Warnings {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
			}
			fmt.Print(graph.FormatTreeAll(result))
		}
		return
	}

	if len(args) > 1 {
		printError(jsonMode, fmt.Sprintf("tree expects at most 1 argument, got %d", len(args)))
		os.Exit(1)
	}

	path := args[0]
	result, err := graph.Tree(repoPath, path, opts)
	if err != nil {
		printError(jsonMode, err.Error())
		os.Exit(1)
	}

	if jsonMode {
		out, _ := json.Marshal(result)
		fmt.Println(string(out))
	} else {
		fmt.Print(graph.FormatTree(result))
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif <command> [--json] [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Manage infrastructure graphs stored in Git.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init <name>       Create a new infrastructure graph")
	fmt.Fprintln(os.Stderr, "  list              List all infrastructure graphs")
	fmt.Fprintln(os.Stderr, "  delete <name>     Delete an infrastructure graph")
	fmt.Fprintln(os.Stderr, "  put <path>        Create or replace a node")
	fmt.Fprintln(os.Stderr, "  get <path>        Read a node")
	fmt.Fprintln(os.Stderr, "  rm <path>         Delete a node")
	fmt.Fprintln(os.Stderr, "  commit [graph]    Commit staged changes")
	fmt.Fprintln(os.Stderr, "  status [graph]    Show uncommitted changes")
	fmt.Fprintln(os.Stderr, "  tree [path]       Show tree hierarchy of a graph")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show help for a command")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'grif <command> --help' for more information on a command.")
}

func printInitUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif init <name> [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Create a new infrastructure graph in the current Git repository.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Creates an empty graph with an orphan commit and a ref at")
	fmt.Fprintln(os.Stderr, "refs/infra/<name>. The current HEAD is recorded as the source commit.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  <name>            Name for the new graph")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif init default")
	fmt.Fprintln(os.Stderr, "  grif init production --json")
}

func printListUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif list [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "List all infrastructure graphs in the current Git repository.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Graphs are listed alphabetically by name.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif list")
	fmt.Fprintln(os.Stderr, "  grif list --json")
}

func printDeleteUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif delete <name> [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Delete an infrastructure graph from the current Git repository.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Removes the graph ref at refs/infra/<name> and any associated")
	fmt.Fprintln(os.Stderr, "staging ref. This operation cannot be undone.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  <name>            Name of the graph to delete")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif delete default")
	fmt.Fprintln(os.Stderr, "  grif delete staging --json")
}

func printGetUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif get <graph/path> [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Read a node from an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "For blob nodes, prints the blob content. For tree nodes, lists")
	fmt.Fprintln(os.Stderr, "the children in a table with type, name, and ID.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  <graph/path>      Path to the node (e.g. default/network/vpc)")
	fmt.Fprintln(os.Stderr, "                    The first segment is the graph name; remaining")
	fmt.Fprintln(os.Stderr, "                    segments address nodes within the graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif get default/network")
	fmt.Fprintln(os.Stderr, "  grif get default/network/vpc")
	fmt.Fprintln(os.Stderr, "  grif get default/network/vpc --json")
}

func printRmUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif rm <graph/path> [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Delete a node from an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Removes the node at the specified path. If the node is a tree,")
	fmt.Fprintln(os.Stderr, "all descendants are removed recursively.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  <graph/path>      Path to the node (e.g. default/network/vpc)")
	fmt.Fprintln(os.Stderr, "                    The first segment is the graph name; remaining")
	fmt.Fprintln(os.Stderr, "                    segments address nodes within the graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif rm default/network/vpc")
	fmt.Fprintln(os.Stderr, "  grif rm default/network              # removes tree and all children")
}

func printCommitUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif commit [graph] [--message <msg>] [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commit staged changes for an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Creates a new commit on the graph's ref with all changes")
	fmt.Fprintln(os.Stderr, "that have been staged via put and rm commands.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "If only one graph exists, the graph name is optional.")
	fmt.Fprintln(os.Stderr, "When multiple graphs exist, the graph name is required.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  [graph]           Name of the graph (optional if only one exists)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --message <msg>   Commit message")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif commit --message \"Add network resources\"")
	fmt.Fprintln(os.Stderr, "  grif commit default --message \"Add network resources\"")
	fmt.Fprintln(os.Stderr, "  grif commit default --json")
}

func printStatusUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif status [graph] [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Show uncommitted changes for an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "If only one graph exists, the graph name is optional.")
	fmt.Fprintln(os.Stderr, "When multiple graphs exist, the graph name is required.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  [graph]           Name of the graph (optional if only one exists)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif status")
	fmt.Fprintln(os.Stderr, "  grif status default")
	fmt.Fprintln(os.Stderr, "  grif status default --json")
}

func printTreeUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif tree [<graph>[/<path>]] [--depth N] [--json]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Show tree hierarchy of an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Displays the recursive tree structure using box-drawing characters.")
	fmt.Fprintln(os.Stderr, "When no argument is given, shows trees for all graphs.")
	fmt.Fprintln(os.Stderr, "When a graph name is given, shows the full tree for that graph.")
	fmt.Fprintln(os.Stderr, "When a path is given (graph/subtree), shows only that subtree.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  [<graph>[/<path>]] Graph name, optionally followed by subtree path")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --depth N         Limit recursion to N levels below root")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif tree")
	fmt.Fprintln(os.Stderr, "  grif tree default")
	fmt.Fprintln(os.Stderr, "  grif tree default/network")
	fmt.Fprintln(os.Stderr, "  grif tree default --depth 1")
	fmt.Fprintln(os.Stderr, "  grif tree --json")
}

func printPutUsage() {
	fmt.Fprintln(os.Stderr, "Usage: grif put <graph/path> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Create or replace a node in an infrastructure graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  <graph/path>      Path to the node (e.g. default/network/vpc)")
	fmt.Fprintln(os.Stderr, "                    The first segment is the graph name; remaining")
	fmt.Fprintln(os.Stderr, "                    segments address nodes within the graph.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Content sources (mutually exclusive):")
	fmt.Fprintln(os.Stderr, "  --data <content>  Provide blob content as an inline string")
	fmt.Fprintln(os.Stderr, "  --file <file>     Read blob content from a file")
	fmt.Fprintln(os.Stderr, "  <stdin>           Pipe content via stdin")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  If no content source is provided, a tree node is created.")
	fmt.Fprintln(os.Stderr, "  Missing intermediate tree nodes are created automatically.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --json            Output in JSON format")
	fmt.Fprintln(os.Stderr, "  -h, --help        Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  grif put default/network/vpc --data \"10.0.0.0/16\"")
	fmt.Fprintln(os.Stderr, "  grif put default/config --file config.json")
	fmt.Fprintln(os.Stderr, "  echo '{\"size\": 10}' | grif put default/storage/disk")
	fmt.Fprintln(os.Stderr, "  grif put default/network              # creates a tree node")
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
		if arg == "--json" || arg == "-h" || arg == "--help" {
			continue
		}
		if arg == "--data" || arg == "--file" || arg == "--message" || arg == "--depth" {
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
