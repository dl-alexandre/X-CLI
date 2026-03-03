package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rodaine/table"
	"github.com/dl-alexandre/go-cli-template/internal/api"
)

// Printer handles output formatting
type Printer struct {
	format   string
	useColor bool
}

// NewPrinter creates a new output printer
func NewPrinter(format string, useColor bool) *Printer {
	return &Printer{
		format:   format,
		useColor: useColor,
	}
}

// PrintItems prints a list of items in the specified format
func (p *Printer) PrintItems(items *api.ListResponse) error {
	switch p.format {
	case "json":
		return p.printJSON(items)
	case "markdown":
		return p.printMarkdown(items)
	case "table":
		return p.printTable(items)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintItem prints a single item in the specified format
func (p *Printer) PrintItem(item *api.Item) error {
	switch p.format {
	case "json":
		return p.printJSON(item)
	case "markdown":
		return p.printItemMarkdown(item)
	case "table":
		return p.printItemTable(item)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printJSON outputs data as formatted JSON
func (p *Printer) printJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// printTable outputs items as a formatted table
func (p *Printer) printTable(items *api.ListResponse) error {
	if len(items.Items) == 0 {
		fmt.Println("No items found.")
		return nil
	}

	tbl := table.New("ID", "Name", "Description", "Created", "Updated").
		WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	for _, item := range items.Items {
		tbl.AddRow(
			item.ID,
			truncate(item.Name, 30),
			truncate(item.Description, 40),
			formatTime(item.CreatedAt),
			formatTime(item.UpdatedAt),
		)
	}

	tbl.Print()
	fmt.Printf("\nShowing %d of %d items\n", len(items.Items), items.Total)

	return nil
}

// printMarkdown outputs items as markdown
func (p *Printer) printMarkdown(items *api.ListResponse) error {
	if len(items.Items) == 0 {
		fmt.Println("No items found.")
		return nil
	}

	fmt.Println("# Items")
	fmt.Println()

	for _, item := range items.Items {
		fmt.Printf("## %s\n\n", item.Name)
		fmt.Printf("**ID:** %s\n\n", item.ID)

		if item.Description != "" {
			fmt.Printf("**Description:** %s\n\n", item.Description)
		}

		fmt.Printf("**Created:** %s\n\n", item.CreatedAt.Format(time.RFC3339))
		fmt.Printf("**Updated:** %s\n\n", item.UpdatedAt.Format(time.RFC3339))

		if len(item.Metadata.Tags) > 0 {
			fmt.Println("**Tags:**")
			for _, tag := range item.Metadata.Tags {
				fmt.Printf("- %s\n", tag)
			}
			fmt.Println()
		}
	}

	fmt.Printf("---\nTotal: %d items\n", items.Total)

	return nil
}

// printItemTable prints a single item as a table
func (p *Printer) printItemTable(item *api.Item) error {
	tbl := table.New("Property", "Value").WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	tbl.AddRow("ID", item.ID)
	tbl.AddRow("Name", item.Name)
	tbl.AddRow("Description", item.Description)
	tbl.AddRow("Created", formatTime(item.CreatedAt))
	tbl.AddRow("Updated", formatTime(item.UpdatedAt))

	if len(item.Metadata.Tags) > 0 {
		tbl.AddRow("Tags", fmt.Sprintf("%v", item.Metadata.Tags))
	}

	if len(item.Metadata.Attributes) > 0 {
		for k, v := range item.Metadata.Attributes {
			tbl.AddRow(k, v)
		}
	}

	tbl.Print()

	return nil
}

// printItemMarkdown prints a single item as markdown
func (p *Printer) printItemMarkdown(item *api.Item) error {
	fmt.Printf("# %s\n\n", item.Name)
	fmt.Printf("**ID:** %s\n\n", item.ID)

	if item.Description != "" {
		fmt.Printf("**Description:** %s\n\n", item.Description)
	}

	fmt.Printf("**Created:** %s\n\n", item.CreatedAt.Format(time.RFC3339))
	fmt.Printf("**Updated:** %s\n\n", item.UpdatedAt.Format(time.RFC3339))

	if len(item.Metadata.Tags) > 0 {
		fmt.Println("**Tags:**")
		for _, tag := range item.Metadata.Tags {
			fmt.Printf("- %s\n", tag)
		}
		fmt.Println()
	}

	if len(item.Metadata.Attributes) > 0 {
		fmt.Println("**Attributes:**")
		for k, v := range item.Metadata.Attributes {
			fmt.Printf("- %s: %s\n", k, v)
		}
		fmt.Println()
	}

	return nil
}

// truncate shortens a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// ValidateFormat checks if a format is supported
func ValidateFormat(format string, allowed []string) error {
	for _, f := range allowed {
		if f == format {
			return nil
		}
	}
	return fmt.Errorf("invalid format '%s', must be one of: %v", format, allowed)
}

// ParseBool parses a boolean string
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
