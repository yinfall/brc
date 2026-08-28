package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DaemonAddr = "127.0.0.1:8082"

type Message struct {
	Type      string `json:"type"`
	PID       int    `json:"pid,omitempty"`
	ID        string `json:"id,omitempty"`
	Code      string `json:"code,omitempty"`
	Success   bool   `json:"success"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ResultRep string `json:"result_repr,omitempty"`
}

type Session struct {
	PID  int
	Conn net.Conn
	Chan map[string]chan Message
	Mu   sync.Mutex
}

var (
	sessions   = make(map[int]*Session)
	sessionsMu sync.Mutex
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "daemon":
		runDaemon()
	case "sessions":
		ensureDaemon()
		runSessions()
	case "exec":
		ensureDaemon()
		runExec(os.Args[2:])
	default:
		if os.Args[1] == "-s" || os.Args[1] == "--session" {
			ensureDaemon()
			runExec(os.Args[1:])
		} else {
			ensureDaemon()
			runExec(append([]string{"exec"}, os.Args[1:]...))
		}
	}
}

func printUsage() {
	fmt.Println("Blender Remote Console CLI (brc)")
	fmt.Println("Usage:")
	fmt.Println("  brc daemon                       Start background daemon")
	fmt.Println("  brc sessions                     List attached Blender sessions")
	fmt.Println("  brc exec <python_code>           Execute Python code")
	fmt.Println("  brc -s <pid> exec <code|file>    Target specific session")
}

func ensureDaemon() {
	conn, err := net.DialTimeout("tcp", DaemonAddr, 50*time.Millisecond)
	if err == nil {
		conn.Close()
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(exe, "daemon")
	cmd.SysProcAttr = nil
	err = cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
		os.Exit(1)
	}
	
	if cmd.Process != nil {
		cmd.Process.Release()
	}

	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err := net.DialTimeout("tcp", DaemonAddr, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Daemon failed to start or bind to %s\n", DaemonAddr)
	os.Exit(1)
}

func runSessions() {
	conn, err := net.Dial("tcp", DaemonAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to daemon: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	msg := Message{Type: "list_sessions"}
	json.NewEncoder(conn).Encode(msg)

	var resp struct {
		Pids []int `json:"pids"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("List of Blender sessions attached")
	if len(resp.Pids) == 0 {
		fmt.Println("No active Blender sessions found.")
	} else {
		for _, pid := range resp.Pids {
			fmt.Printf("%d\tdevice (Blender Remote Console)\n", pid)
		}
	}
}

func runExec(args []string) {
	var targetPid int
	var code string

	i := 0
	for i < len(args) {
		if args[i] == "-s" || args[i] == "--session" {
			if i+1 < len(args) {
				pid, _ := strconv.Atoi(args[i+1])
				targetPid = pid
				i += 2
			} else {
				fmt.Fprintln(os.Stderr, "Missing PID after -s")
				os.Exit(1)
			}
		} else if args[i] == "exec" {
			i++
		} else {
			break
		}
	}

	if i < len(args) {
		arg := strings.Join(args[i:], " ")
		if info, err := os.Stat(arg); err == nil && !info.IsDir() {
			content, _ := os.ReadFile(arg)
			code = string(content)
		} else {
			code = arg
		}
	}

	if strings.TrimSpace(code) == "" {
		fmt.Fprintln(os.Stderr, "brc: error: no Python code or file specified.")
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", DaemonAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to daemon: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	msg := Message{
		Type: "cli_exec",
		PID:  targetPid,
		Code: code,
	}
	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	var resp Message
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error receiving response: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}
}

func runDaemon() {
	l, err := net.Listen("tcp", DaemonAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Daemon could not bind to %s: %v\n", DaemonAddr, err)
		os.Exit(1)
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

	case "list_sessions":
		defer conn.Close()
		sessionsMu.Lock()
		pids := make([]int, 0, len(sessions))
		for pid := range sessions {
			pids = append(pids, pid)
		}
		sessionsMu.Unlock()
		json.NewEncoder(conn).Encode(map[string]interface{}{"pids": pids})

	case "cli_exec":
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
	default:
		conn.Close()
	}
}
