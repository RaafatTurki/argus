package main

import (
	"fmt"
	"os"

	"github.com/RaafatTurki/argus/internal/keys"
	"github.com/RaafatTurki/argus/internal/tui"
)

const usage = `argus          open the interface
argus keys     print the key bindings
`

func main() {
	km := keys.Load()

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "":
		if err := tui.Run(km); err != nil {
			fmt.Fprintln(os.Stderr, "argus:", err)
			os.Exit(1)
		}
	case "keys":
		fmt.Print(km.Dump())
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "argus: unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
}
