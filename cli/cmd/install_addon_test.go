package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectBlenderTargets(t *testing.T) {
	tempDir := t.TempDir()

	// Mock environments for all OSes to make the test cross-platform
	t.Setenv("APPDATA", tempDir) // Windows
	t.Setenv("HOME", tempDir)    // macOS/Linux

	// Create Windows mock
	winDir42 := filepath.Join(tempDir, "Blender Foundation", "Blender", "4.2", "scripts", "addons")
	winDir50 := filepath.Join(tempDir, "Blender Foundation", "Blender", "5.0", "scripts", "addons")
	winDir36 := filepath.Join(tempDir, "Blender Foundation", "Blender", "3.6", "scripts", "addons") // Should be included
	winDir279 := filepath.Join(tempDir, "Blender Foundation", "Blender", "2.79", "scripts", "addons") // Should be ignored (< 2.80)
	os.MkdirAll(winDir42, 0755)
	os.MkdirAll(winDir50, 0755)
	os.MkdirAll(winDir36, 0755)
	os.MkdirAll(winDir279, 0755)

	// Create macOS mock
	macDir42 := filepath.Join(tempDir, "Library", "Application Support", "Blender", "4.2", "scripts", "addons")
	macDir50 := filepath.Join(tempDir, "Library", "Application Support", "Blender", "5.0", "scripts", "addons")
	macDir36 := filepath.Join(tempDir, "Library", "Application Support", "Blender", "3.6", "scripts", "addons")
	macDir279 := filepath.Join(tempDir, "Library", "Application Support", "Blender", "2.79", "scripts", "addons")
	os.MkdirAll(macDir42, 0755)
	os.MkdirAll(macDir50, 0755)
	os.MkdirAll(macDir36, 0755)
	os.MkdirAll(macDir279, 0755)

	// Create Linux mock (.config/blender)
	linDir42 := filepath.Join(tempDir, ".config", "blender", "4.2", "scripts", "addons")
	linDir50 := filepath.Join(tempDir, ".config", "blender", "5.0", "scripts", "addons")
	linDir36 := filepath.Join(tempDir, ".config", "blender", "3.6", "scripts", "addons")
	linDir279 := filepath.Join(tempDir, ".config", "blender", "2.79", "scripts", "addons")
	os.MkdirAll(linDir42, 0755)
	os.MkdirAll(linDir50, 0755)
	os.MkdirAll(linDir36, 0755)
	os.MkdirAll(linDir279, 0755)

	targets := detectBlenderTargets()

	expectedCount := 3 // We created 3.6, 4.2 and 5.0 for each OS path

	// Linux actually checks multiple base dirs, but only .config/blender is populated here.
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		// It will find the .config ones
		expectedCount = 3
	}

	if len(targets) != expectedCount {
		t.Fatalf("expected %d valid targets, got %d", expectedCount, len(targets))
	}

	// Should be sorted by version (ascending: 3.6, 4.2 then 5.0)
	if targets[0].Major != 3 || targets[0].Minor != 6 {
		t.Errorf("expected first target to be 3.6, got %d.%d", targets[0].Major, targets[0].Minor)
	}
	if targets[1].Major != 4 || targets[1].Minor != 2 {
		t.Errorf("expected second target to be 4.2, got %d.%d", targets[1].Major, targets[1].Minor)
	}
	if targets[2].Major != 5 || targets[2].Minor != 0 {
		t.Errorf("expected third target to be 5.0, got %d.%d", targets[2].Major, targets[2].Minor)
	}
}
