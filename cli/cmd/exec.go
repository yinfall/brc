package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var targetPid int

var execCmd = &cobra.Command{
	Use:   "exec [python_code_or_file]",
	Short: "Execute Python code in active Blender",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemon(); err != nil {
			return err
		}
		var code string

		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, _ := io.ReadAll(os.Stdin)
			code = string(bytes)
		}

		if code == "" && len(args) > 0 {
			arg := strings.Join(args, " ")
			if info, err := os.Stat(arg); err == nil && !info.IsDir() {
				content, _ := os.ReadFile(arg)
				code = string(content)
			} else {
				code = arg
			}
		}

		if strings.TrimSpace(code) == "" {
			return fmt.Errorf("no Python code or file specified")
		}

		conn, err := net.Dial("tcp", DaemonAddr)
		if err != nil {
			return fmt.Errorf("error connecting to daemon: %w", err)
		}
		defer conn.Close()

		msg := Message{
			Type: "cli_exec",
			PID:  targetPid,
			Code: code,
		}
		if err := json.NewEncoder(conn).Encode(msg); err != nil {
			return fmt.Errorf("error sending request: %w", err)
		}

		var resp Message
		if err := json.NewDecoder(conn).Decode(&resp); err != nil {
			return fmt.Errorf("error receiving response: %w", err)
		}

		if resp.Stdout != "" {
			fmt.Fprint(os.Stdout, resp.Stdout)
		}
		if resp.Stderr != "" {
			fmt.Fprint(os.Stderr, resp.Stderr)
		}
		if resp.ResultRep != "" {
			fmt.Fprintln(os.Stdout, resp.ResultRep)
		}
		if !resp.Success {
			return fmt.Errorf("execution failed")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
	rootCmd.PersistentFlags().IntVarP(&targetPid, "session", "s", 0, "Target specific Blender session by PID")
}
