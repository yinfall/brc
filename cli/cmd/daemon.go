package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start background daemon server directly",
	RunE: func(cmd *cobra.Command, args []string) error {
		l, err := net.Listen("tcp", DaemonAddr)
		if err != nil {
			return fmt.Errorf("daemon could not bind to %s: %w", DaemonAddr, err)
		}
		defer l.Close()

		fmt.Printf("Daemon listening on %s\n", DaemonAddr)

		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Printf("Accept error: %v\n", err)
				continue
			}
			go handleConnection(conn)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		conn.Close()
		return
	}

	var firstMsg Message
	if err := json.Unmarshal(line, &firstMsg); err != nil {
		conn.Close()
		return
	}

	switch firstMsg.Type {
	case "register":
		handleRegister(conn, reader, firstMsg)
	case "list_sessions":
		handleListSessions(conn)
	case "cli_exec":
		handleCliExec(conn, firstMsg)
	default:
		conn.Close()
	}
}

func handleRegister(conn net.Conn, reader *bufio.Reader, firstMsg Message) {
	pid := firstMsg.PID
	if pid <= 0 {
		conn.Close()
		return
	}

	session := &Session{
		PID:  pid,
		Conn: conn,
		Chan: make(map[string]chan Message),
	}

	sessionsMu.Lock()
	if old, ok := sessions[pid]; ok {
		old.Conn.Close()
	}
	sessions[pid] = session
	sessionsMu.Unlock()
	fmt.Printf("Registered session PID %d\n", pid)

	go func() {
		defer func() {
			conn.Close()
			sessionsMu.Lock()
			if sessions[pid] == session {
				delete(sessions, pid)
			}
			sessionsMu.Unlock()
			fmt.Printf("Session PID %d disconnected\n", pid)
		}()

		for {
			b, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var msg Message
			if err := json.Unmarshal(b, &msg); err != nil {
				continue
			}
			if msg.Type == "result" {
				session.Mu.Lock()
				ch, ok := session.Chan[msg.ID]
				if ok {
					delete(session.Chan, msg.ID)
				}
				session.Mu.Unlock()
				if ok {
					ch <- msg
				}
			}
		}
	}()
}

func handleListSessions(conn net.Conn) {
	defer conn.Close()
	sessionsMu.Lock()
	pids := make([]int, 0, len(sessions))
	for pid := range sessions {
		pids = append(pids, pid)
	}
	sessionsMu.Unlock()
	json.NewEncoder(conn).Encode(map[string]interface{}{"pids": pids})
}

func handleCliExec(conn net.Conn, firstMsg Message) {
	defer conn.Close()
	targetPid := firstMsg.PID
	sessionsMu.Lock()

	var session *Session
	if targetPid > 0 {
		session = sessions[targetPid]
	} else if len(sessions) == 1 {
		for _, s := range sessions {
			session = s
		}
	} else if len(sessions) > 1 {
		sessionsMu.Unlock()
		json.NewEncoder(conn).Encode(Message{
			Success: false,
			Stderr:  "Error: more than one Blender session active. Use -s <pid>.\n",
		})
		return
	} else {
		sessionsMu.Unlock()
		json.NewEncoder(conn).Encode(Message{
			Success: false,
			Stderr:  "Error: no Blender sessions found.\n",
		})
		return
	}
	sessionsMu.Unlock()

	if session == nil {
		json.NewEncoder(conn).Encode(Message{
			Success: false,
			Stderr:  fmt.Sprintf("Error: session %d not found.\n", targetPid),
		})
		return
	}

	reqId := fmt.Sprintf("%d", time.Now().UnixNano())
	reqMsg := Message{
		Type: "exec",
		ID:   reqId,
		Code: firstMsg.Code,
	}

	ch := make(chan Message, 1)
	session.Mu.Lock()
	session.Chan[reqId] = ch
	session.Mu.Unlock()

	reqMsgBytes, _ := json.Marshal(reqMsg)
	reqMsgBytes = append(reqMsgBytes, '\n')

	if _, err := session.Conn.Write(reqMsgBytes); err != nil {
		session.Mu.Lock()
		delete(session.Chan, reqId)
		session.Mu.Unlock()
		json.NewEncoder(conn).Encode(Message{
			Success: false,
			Stderr:  fmt.Sprintf("Error sending to plugin: %v\n", err),
		})
		return
	}

	select {
	case resp := <-ch:
		json.NewEncoder(conn).Encode(resp)
	case <-time.After(60 * time.Second):
		session.Mu.Lock()
		delete(session.Chan, reqId)
		session.Mu.Unlock()
		json.NewEncoder(conn).Encode(Message{
			Success: false,
			Stderr:  "Error: Execution timed out (60s).\n",
		})
	}
}
