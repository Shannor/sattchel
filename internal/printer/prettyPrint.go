package printer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/chroma/v2/quick"
)

// PrettyPrintColor takes any struct or variable, converts it to JSON,
// and prints it to the terminal with syntax highlighting.
func PrettyPrintColor(v any) {
	// 1. Convert the struct to formatted JSON bytes
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error formatting struct:", err)
		return
	}

	// 2. Convert bytes to string
	jsonString := string(b)

	// 3. Use Chroma to highlight the JSON string and write it directly to standard output
	// Parameters:
	// - os.Stdout: Where to print it
	// - jsonString: The text to highlight
	// - "json": The language lexer to use (tells chroma how to parse it)
	// - "terminal256": The formatter (tells chroma to use terminal escape codes)
	// - "monokai": The color theme (you can change this to "dracula", "github", etc.)
	err = quick.Highlight(os.Stdout, jsonString, "json", "terminal256", "monokai")
	if err != nil {
		// Fallback to normal print if highlighting fails
		fmt.Println(jsonString)
	}

	// Add a final newline so the next terminal prompt isn't on the same line
	fmt.Println()
}
