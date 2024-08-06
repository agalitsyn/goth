package flagtools

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var Prefix = ""

// Parse will set each defined flag from its corresponding environment variable with Prefix and next without it.
// If dots or dash are presents in the flag name, they will be converted to underscores.
// If Parse fails, a fatal error is issued.
func Parse() {
	if err := ParseSet(Prefix, flag.CommandLine); err != nil {
		log.Fatalln(err)
	}
}

// ParseSet parses the given flagset. The specified prefix will be applied to
// the environment variable names.
func ParseSet(prefix string, set *flag.FlagSet) error {
	var explicit []*flag.Flag
	var all []*flag.Flag
	set.Visit(func(f *flag.Flag) {
		explicit = append(explicit, f)
	})

	var err error
	set.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}

		all = append(all, f)
		if !contains(explicit, f) {
			name := strings.Replace(f.Name, ".", "_", -1)
			name = strings.Replace(name, "-", "_", -1)
			name = strings.ToUpper(name)

			var prefixedName string
			toScan := []string{name}
			if prefix != "" {
				prefixedName = fmt.Sprintf("%s_%s", prefix, name)
				prefixedName = strings.ToUpper(prefixedName)
				// append the prefixed name to the list of env vars in the beginning, have more priority
				toScan = append([]string{prefixedName}, toScan...)
			}

			for _, envVar := range toScan {
				value, ok := os.LookupEnv(envVar)
				if ok {
					if ferr := f.Value.Set(value); ferr != nil {
						err = fmt.Errorf("failed to set flag %q with value %q", f.Name, value)
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
	})
	return err
}

func contains(list []*flag.Flag, f *flag.Flag) bool {
	for _, i := range list {
		if i == f {
			return true
		}
	}
	return false
}
