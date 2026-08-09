package cli

import (
	"encoding/json"
	"fmt"
)

func (a App) writeJSON(value any) int {
	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(a.stderr, "sesa: encode JSON output: %v\n", err)
		return 1
	}
	return 0
}
