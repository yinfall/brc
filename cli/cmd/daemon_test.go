package cmd

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// setupTestDaemon starts the daemon on a random open port and updates DaemonAddr.
// It returns a cleanup function to close the listener and reset state.
func setupTestDaemon(t *testing.T) (net.Listener, func()) {
	// Reset global state
	sessionsMu.Lock()
	sessions = make(map[int]*Session)
	sessionsMu.Unlock()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Update DaemonAddr so handleCliExec and tests use the dynamic port
	originalDaemonAddr := DaemonAddr
	DaemonAddr = l.Addr().String()

	// Start accepting connections in background
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return // Closed
			}
			go handleConnection(conn)
		}
	}()

	cleanup := func() {
		l.Close()
		DaemonAddr = originalDaemonAddr
		sessionsMu.Lock()
		for _, s := range sessions {
			s.Conn.Close()
		}
		sessions = make(map[int]*Session)
		sessionsMu.Unlock()
	}

	return l, cleanup
}

func connectAndSend(t *testing.T, msg Message) net.Conn {
	conn, err := net.Dial("tcp", DaemonAddr)
	if err != nil {
		t.Fatalf("Failed to connect to test daemon: %v", err)
	}
	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	// Append newline to ensure bufio.Reader triggers
	conn.Write([]byte("\n"))
	return conn
}

func TestDaemon_SingleSessionExecution(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	// 1. Mock Blender connects and registers
	blenderConn := connectAndSend(t, Message{Type: "register", PID: 1001})
	defer blenderConn.Close()

	// Give daemon a tiny moment to register the session
	time.Sleep(50 * time.Millisecond)

	sessionsMu.Lock()
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}
	sessionsMu.Unlock()

	// 2. Mock CLI sends exec request in background
	cliReqMsg := Message{Type: "cli_exec", Code: "print('hello')", PID: 0}
	
	// Start a goroutine for Blender to wait for exec and reply
	go func() {
		reader := bufio.NewReader(blenderConn)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		var req Message
		json.Unmarshal(line, &req)
		
		if req.Type == "exec" && req.Code == "print('hello')" {
			// Send fake result back
			res := Message{
				Type:    "result",
				ID:      req.ID,
				Success: true,
				Stdout:  "hello\n",
			}
			json.NewEncoder(blenderConn).Encode(res)
			blenderConn.Write([]byte("\n"))
		}
	}()

	// 3. Mock CLI connects and sends execution request
	cliConn := connectAndSend(t, cliReqMsg)
	defer cliConn.Close()

	// 4. CLI should receive result
	cliReader := bufio.NewReader(cliConn)
	resLine, err := cliReader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("CLI failed to read response: %v", err)
	}

	var res Message
	if err := json.Unmarshal(resLine, &res); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !res.Success || res.Stdout != "hello\n" {
		t.Errorf("Unexpected result: %+v", res)
	}
}

func TestDaemon_MultipleSessionsError(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Register 2 Blenders
	b1 := connectAndSend(t, Message{Type: "register", PID: 1001})
	defer b1.Close()
	b2 := connectAndSend(t, Message{Type: "register", PID: 1002})
	defer b2.Close()
	
	time.Sleep(50 * time.Millisecond)

	// CLI sends exec without targeting specific PID
	cliConn := connectAndSend(t, Message{Type: "cli_exec", Code: "print('hello')"})
	defer cliConn.Close()

	cliReader := bufio.NewReader(cliConn)
	resLine, _ := cliReader.ReadBytes('\n')

	var res Message
	json.Unmarshal(resLine, &res)

	if res.Success || res.Stderr == "" {
		t.Errorf("Expected error for multiple sessions, got success. Res: %+v", res)
	}
}

func TestDaemon_SmartWait(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	// CLI sends exec FIRST (while zero sessions exist)
	cliConn := connectAndSend(t, Message{Type: "cli_exec", Code: "print('delayed')"})
	defer cliConn.Close()

	// Wait 0.5s, then Blender registers (within the 2.5s window)
	time.Sleep(500 * time.Millisecond)

	blenderConn := connectAndSend(t, Message{Type: "register", PID: 9999})
	defer blenderConn.Close()

	go func() {
		reader := bufio.NewReader(blenderConn)
		line, _ := reader.ReadBytes('\n')
		var req Message
		json.Unmarshal(line, &req)
		
		res := Message{Type: "result", ID: req.ID, Success: true, Stdout: "delayed success\n"}
		json.NewEncoder(blenderConn).Encode(res)
		blenderConn.Write([]byte("\n"))
	}()

	cliReader := bufio.NewReader(cliConn)
	resLine, err := cliReader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("CLI read error: %v", err)
	}

	var res Message
	json.Unmarshal(resLine, &res)
	if !res.Success || res.Stdout != "delayed success\n" {
		t.Errorf("Smart wait failed, unexpected result: %+v", res)
	}
}

func TestDaemon_SmartWaitTimeout(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	start := time.Now()
	// CLI sends exec, but NO Blender ever connects
	cliConn := connectAndSend(t, Message{Type: "cli_exec", Code: "print('fail')"})
	defer cliConn.Close()

	cliReader := bufio.NewReader(cliConn)
	resLine, _ := cliReader.ReadBytes('\n')

	duration := time.Since(start)

	var res Message
	json.Unmarshal(resLine, &res)

	if res.Success || res.Stderr != "Error: no Blender sessions found.\n" {
		t.Errorf("Expected no sessions error, got: %+v", res)
	}

	// Should have waited approx 2.5s (25 * 100ms = 2.5s)
	if duration < 2400*time.Millisecond || duration > 3000*time.Millisecond {
		t.Errorf("Expected wait around 2.5s, got %v", duration)
	}
}

func TestDaemon_ListSessions(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Register 2 Blenders
	b1 := connectAndSend(t, Message{Type: "register", PID: 5001})
	defer b1.Close()
	b2 := connectAndSend(t, Message{Type: "register", PID: 5002})
	defer b2.Close()
	
	time.Sleep(50 * time.Millisecond)

	cliConn := connectAndSend(t, Message{Type: "list_sessions"})
	defer cliConn.Close()

	cliReader := bufio.NewReader(cliConn)
	resLine, _ := cliReader.ReadBytes('\n')

	var res struct {
		Pids []int `json:"pids"`
	}
	json.Unmarshal(resLine, &res)

	if len(res.Pids) != 2 {
		t.Errorf("Expected 2 PIDs, got %d: %v", len(res.Pids), res.Pids)
	}
}

func TestDaemon_RegisterOverwrite(t *testing.T) {
	_, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Register first Blender
	b1 := connectAndSend(t, Message{Type: "register", PID: 777})
	time.Sleep(50 * time.Millisecond)

	sessionsMu.Lock()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session")
	}
	sessionsMu.Unlock()

	// Register second Blender with SAME PID
	b2 := connectAndSend(t, Message{Type: "register", PID: 777})
	defer b2.Close()
	
	time.Sleep(50 * time.Millisecond)

	sessionsMu.Lock()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session after overwrite, got %d", len(sessions))
	}
	sessionsMu.Unlock()

	// Ensure b1 was closed (read should fail)
	b1.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	_, err := b1.Read(make([]byte, 1))
	if err == nil {
		t.Errorf("Expected old connection to be closed")
	}
}
