package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetRootCommandState() {
	resetFlags := func(flags *pflag.FlagSet) {
		flags.VisitAll(func(flag *pflag.Flag) {
			if replacer, ok := flag.Value.(interface{ Replace([]string) error }); ok {
				_ = replacer.Replace([]string{})
			} else {
				_ = flags.Set(flag.Name, flag.DefValue)
			}
			flag.Changed = false
		})
	}

	var reset func(*cobra.Command)
	reset = func(cmd *cobra.Command) {
		cmd.SetArgs(nil)
		resetFlags(cmd.Flags())
		resetFlags(cmd.PersistentFlags())
		for _, sub := range cmd.Commands() {
			reset(sub)
		}
	}

	cfg = rootConfig{}
	versionFlag = false
	reset(rootCmd)
}
