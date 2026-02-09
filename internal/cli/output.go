package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// outputFormat holds the current output format setting.
var outputFormat string

// Structured result for AI-friendly output.
type result struct {
	OK        bool        `json:"ok"`
	Command   string      `json:"command"`
	Data      interface{} `json:"data"`
	Summary   string      `json:"summary,omitempty"`
	NextSteps []string    `json:"next_steps,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// printResult outputs data in the chosen format (table/json/ai).
func printResult(command string, data interface{}, summary string, nextSteps []string) {
	switch outputFormat {
	case "json":
		printJSON(command, data, summary, nextSteps)
	case "ai":
		printAI(command, data, summary, nextSteps)
	default:
		// table format is handled by callers; this is fallback
		printAI(command, data, summary, nextSteps)
	}
}

func printJSON(command string, data interface{}, summary string, nextSteps []string) {
	r := result{
		OK:        true,
		Command:   command,
		Data:      data,
		Summary:   summary,
		NextSteps: nextSteps,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(r)
}

func printAI(command string, data interface{}, summary string, nextSteps []string) {
	fmt.Printf("## %s\n\n", command)
	if summary != "" {
		fmt.Printf("%s\n\n", summary)
	}

	// Pretty-print data as indented JSON for AI readability
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("```json\n%s\n```\n", string(b))

	if len(nextSteps) > 0 {
		fmt.Printf("\n**Next steps:**\n")
		for _, s := range nextSteps {
			fmt.Printf("- %s\n", s)
		}
	}
	fmt.Println()
}

func printErrorResult(command string, err error, hint string) {
	if outputFormat == "json" {
		r := result{
			OK:        false,
			Command:   command,
			Summary:   err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if hint != "" {
			r.NextSteps = []string{hint}
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(r)
		return
	}

	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	if hint != "" {
		fmt.Fprintf(os.Stderr, "\n  Hint: %s\n", hint)
	}
}

// renderTable creates and renders a table to stdout.
func renderTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("  (no results)")
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetAutoFormatHeaders(false)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}

// isFormatJSON returns true if output is json or ai.
func isStructuredOutput() bool {
	return outputFormat == "json" || outputFormat == "ai"
}

// printSuccess prints a green success message (table mode only).
func printSuccess(msg string) {
	color.New(color.FgGreen, color.Bold).Println(msg)
}

// printSection prints a bold section header.
func printSection(title string) {
	fmt.Println()
	color.New(color.Bold, color.FgCyan).Println(title)
	fmt.Println(strings.Repeat("-", len(title)+4))
}

// printKeyValue prints a formatted key-value pair.
func printKeyValue(key, value string) {
	bold := color.New(color.Bold)
	bold.Printf("  %-14s ", key+":")
	fmt.Println(value)
}
