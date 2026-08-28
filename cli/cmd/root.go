package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	SilenceUsage:  true,
	SilenceErrors: true,
	Use:   "brc",
	Short: "Blender Remote Console CLI",
	Long:  `Blender Remote Console CLI (brc) - A tool to execute Python code remotely in Blender.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Fallback for stdin piping without 'exec'
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Pass to exec logic
			return execCmd.RunE(execCmd, args)
		}
		cmd.Help()
		return nil
	},
}

func prepareArgs(args []string) []string {
	// Dynamically collect known commands
	knownCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		knownCommands[cmd.Name()] = true
		for _, alias := range cmd.Aliases {
			knownCommands[alias] = true
		}
	}
	knownCommands["help"] = true
	knownCommands["completion"] = true

	hasKnownCmd := false
	for _, arg := range args[1:] {
		if knownCommands[arg] {
			hasKnownCmd = true
			break
		}
	}

	if !hasKnownCmd && len(args) > 1 {
		shouldInjectExec := false
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			
			// Heuristic 1: is it a Python file?
			if strings.HasSuffix(arg, ".py") {
				shouldInjectExec = true
				break
			}
			// Heuristic 2: does it look like Python code?
			if strings.ContainsAny(arg, "()=\"' ") {
				shouldInjectExec = true
				break
			}
			// Heuristic 3: is it a local file?
			if info, err := os.Stat(arg); err == nil && !info.IsDir() {
				shouldInjectExec = true
				break
			}
		}

		if shouldInjectExec {
			// Inject "exec" at the first position after the binary name
			return append([]string{args[0], "exec"}, args[1:]...)
		}
	}
	return args
}

func Execute() {
	os.Args = prepareArgs(os.Args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
