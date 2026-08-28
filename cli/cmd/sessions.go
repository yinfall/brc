package cmd

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List attached Blender sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemon(); err != nil {
			return err
		}
		conn, err := net.Dial("tcp", DaemonAddr)
		if err != nil {
			return fmt.Errorf("error connecting to daemon: %w", err)
		}
		defer conn.Close()

		msg := Message{Type: "list_sessions"}
		json.NewEncoder(conn).Encode(msg)

		var resp struct {
			Pids []int `json:"pids"`
		}
		if err := json.NewDecoder(conn).Decode(&resp); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}

		fmt.Println("List of Blender sessions attached")
		if len(resp.Pids) == 0 {
			fmt.Println("No active Blender sessions found.")
		} else {
			for _, pid := range resp.Pids {
				fmt.Printf("%d\tdevice (Blender Remote Console)\n", pid)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
}
