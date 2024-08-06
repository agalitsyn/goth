package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func completionFromStaticVariants(cmd *cobra.Command, flag string, variants ...string) {
	completionFromVariants(cmd, flag, func() []string {
		return variants
	})
}

func completionFromVariants(cmd *cobra.Command, flag string, fn func() []string) {
	err := cmd.RegisterFlagCompletionFunc(
		flag,
		completionFnFromVariants(func(_ []string) []string {
			return fn()
		}),
	)
	if err != nil {
		panic(fmt.Errorf("unable to register completion func: %w", err))
	}
}

func completionFnFromVariants(
	fn func(args []string) []string,
) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		variants := fn(args)
		result := make([]string, 0, len(variants))
		for _, v := range variants {
			if strings.HasPrefix(v, toComplete) {
				result = append(result, v)
			}
		}
		return result, cobra.ShellCompDirectiveNoFileComp
	}
}

func MustMarkFlagRequired(cmd *cobra.Command, name string) {
	err := cmd.MarkFlagRequired(name)
	if err != nil {
		panic(fmt.Sprintf("unable to mark flag required: %v", err))
	}
}
