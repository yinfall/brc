package cmd

import (
	"net"
	"os"
	"os/exec"
	"fmt"
	"time"
	"sync"
)

var DaemonAddr = "127.0.0.1:8082"
const Version = "v0.1.0-alpha.1"

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

func ensureDaemon() error {
	conn, err := net.DialTimeout("tcp", DaemonAddr, 50*time.Millisecond)
	if err == nil {
		conn.Close()
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	fmt.Fprintf(os.Stderr, "* daemon not running; starting now at %s\n", DaemonAddr)
	c := exec.Command(exe, "daemon")
	c.SysProcAttr = nil
	err = c.Start()
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	if c.Process != nil {
		c.Process.Release()
	}

	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err := net.DialTimeout("tcp", DaemonAddr, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			fmt.Fprintln(os.Stderr, "* daemon started successfully")
			return nil
		}
	}
	return fmt.Errorf("daemon failed to start or bind to %s", DaemonAddr)
}
