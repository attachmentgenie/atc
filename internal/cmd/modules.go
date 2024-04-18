package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/attachmentgenie/atc/pkg/atc"
)

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "List available values that can be used as target.",
	Long:  "List available values that can be used as target.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := atc.Config{}
		_ = cfg.Server.LogLevel.Set("info")
		t, _ := atc.New(cfg)
		allDeps := t.ModuleManager.DependenciesForModule(atc.All)

		for _, m := range t.ModuleManager.UserVisibleModuleNames() {
			ix := sort.SearchStrings(allDeps, m)
			included := ix < len(allDeps) && allDeps[ix] == m

			if included {
				fmt.Fprintln(os.Stdout, m, "*")
			} else {
				fmt.Fprintln(os.Stdout, m)
			}
		}

		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Modules marked with * are included in target All.")
	},
}

func init() {
	rootCmd.AddCommand(modulesCmd)
}
