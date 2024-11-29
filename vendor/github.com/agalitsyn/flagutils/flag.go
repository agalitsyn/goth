package flagutils

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	// Prefix is used to prefix environment variables.
	Prefix = ""
)

// Parse parses flags from environment variables.
func Parse() {
	flag.VisitAll(func(f *flag.Flag) {
		visitor(flag.CommandLine, f)
	})
}

// ParseFlagSet parses flags from environment variables for the given FlagSet.
func ParseFlagSet(fs *flag.FlagSet) {
	fs.VisitAll(func(f *flag.Flag) {
		visitor(fs, f)
	})
}

func visitor(fs *flag.FlagSet, f *flag.Flag) {
	name := strings.ToUpper(f.Name)
	name = strings.ReplaceAll(name, "-", "_")

	var prefixedName string
	toScan := []string{name}
	if Prefix != "" {
		prefixedName = fmt.Sprintf("%s_%s", Prefix, name)
		prefixedName = strings.ToUpper(prefixedName)
		// append the prefixed name to the list of env vars in the beginning, have more priority
		toScan = append([]string{prefixedName}, toScan...)
	}

	for _, envName := range toScan {
		value, ok := os.LookupEnv(envName)
		if ok {
			if err := fs.Set(f.Name, value); err != nil {
				fmt.Fprintf(os.Stderr, "invalid value %q for environment variable %s: %s\n", value, envName, err)
				os.Exit(2)
			}
			break
		}
	}

	if prefixedName != "" {
		f.Usage += fmt.Sprintf(" [ENV: %s or %s].", prefixedName, name)
	} else {
		f.Usage += fmt.Sprintf(" [ENV: %s].", name)
	}
}
