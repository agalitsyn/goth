package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func MustMarkFlagRequired(cmd *cobra.Command, name string) {
	err := cmd.MarkFlagRequired(name)
	if err != nil {
		panic(fmt.Sprintf("unable to mark flag required: %v", err))
	}
}
