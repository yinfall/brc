package cmd

import (
	"archive/zip"
	"bufio"
	"github.com/spf13/cobra"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	AddonFolderName = "blender-remote-console"
	ZipFileName     = "blender-remote-console.zip"
)

type BlenderTarget struct {
	VersionName string
	Major       int
	Minor       int
	AddonDir    string
}

// detectBlenderTargets finds all standard Blender 4.x+ addon directories on the current system
func detectBlenderTargets() []BlenderTarget {
	var results []BlenderTarget
	var baseDirs []string

	homeDir, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			baseDirs = append(baseDirs, filepath.Join(appData, "Blender Foundation", "Blender"))
		}
	case "darwin":
		if homeDir != "" {
			baseDirs = append(baseDirs, filepath.Join(homeDir, "Library", "Application Support", "Blender"))
		}
	default: // linux and others
		if homeDir != "" {
			baseDirs = append(baseDirs,
				filepath.Join(homeDir, ".config", "blender"),
				filepath.Join(homeDir, ".var", "app", "org.blender.Blender", "config", "blender"),
				filepath.Join(homeDir, "snap", "blender", "current", ".config", "blender"),
			)
		}
	}

	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)$`)

	for _, base := range baseDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				matches := versionRegex.FindStringSubmatch(entry.Name())
				if len(matches) == 3 {
					major, _ := strconv.Atoi(matches[1])
					minor, _ := strconv.Atoi(matches[2])
					// Support Blender 4.0+
					if major >= 4 {
						addonDir := filepath.Join(base, entry.Name(), "scripts", "addons")
						results = append(results, BlenderTarget{
							VersionName: fmt.Sprintf("Blender %s", entry.Name()),
							Major:       major,
							Minor:       minor,
							AddonDir:    addonDir,
						})
					}
				}
			}
		}
	}

	// Sort by version (ascending)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Major != results[j].Major {
			return results[i].Major < results[j].Major
		}
		return results[i].Minor < results[j].Minor
	})

	return results
}

// findLocalZip tries to locate blender-remote-console.zip in standard local paths
func findLocalZip(customZip string) (string, error) {
	if customZip != "" {
		if _, err := os.Stat(customZip); err == nil {
			return customZip, nil
		}
		return "", fmt.Errorf("specified zip file not found: %s", customZip)
	}

	homeDir, _ := os.UserHomeDir()
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	candidatePaths := []string{
		filepath.Join(exeDir, ZipFileName),
		filepath.Join(exeDir, "..", ZipFileName),
		filepath.Join(homeDir, ".brc", ZipFileName),
		filepath.Join(homeDir, ".brc", "bin", ZipFileName),
		ZipFileName, // current working directory
	}

	for _, p := range candidatePaths {
		cleanP := filepath.Clean(p)
		if info, err := os.Stat(cleanP); err == nil && !info.IsDir() {
			return cleanP, nil
		}
	}

	return "", fmt.Errorf("could not find %s in ~/.brc/ or next to brc binary", ZipFileName)
}

// extractZipFile extracts a zip archive directly to destAddonsDir/blender-remote-console
func extractZipFile(zipPath string, destAddonsDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file %s: %w", zipPath, err)
	}
	defer reader.Close()

	targetAddonDir := filepath.Join(destAddonsDir, AddonFolderName)
	if err := os.MkdirAll(targetAddonDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target addon directory %s: %w", targetAddonDir, err)
	}

	for _, f := range reader.File {
		cleanName := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "\\") {
			continue // Zip Slip protection
		}

		targetPath := filepath.Join(targetAddonDir, cleanName)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", err
		}

		srcFile, err := f.Open()
		if err != nil {
			return "", err
		}

		dstFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			srcFile.Close()
			return "", err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
		if err != nil {
			return "", err
		}
	}

	return targetAddonDir, nil
}

func runInstallAddon(customPath string, customZip string, allVersions bool) error {

	fmt.Println(">>> Blender Remote Console Addon Installation")
	fmt.Println()

	// 1. Locate the local zip file
	zipPath, err := findLocalZip(customZip)
	if err != nil {
		return fmt.Errorf("❌ Error: %v\n👉 Please ensure 'blender-remote-console.zip' is placed in ~/.brc/ or use:\n   brc install-addon --zip /path/to/blender-remote-console.zip", err)
	}
	fmt.Printf("✓ Found addon package: %s\n\n", zipPath)

	// 2. Custom path mode
	if customPath != "" {
		installedDir, err := extractZipFile(zipPath, customPath)
		if err != nil {
			return fmt.Errorf("❌ Failed to install to %s: %v", customPath, err)
		}
		fmt.Printf("✓ Successfully installed addon to: %s\n", installedDir)
		printNextSteps()
		return nil
	}

	// 3. Scan detected installations
	targets := detectBlenderTargets()
	if len(targets) == 0 {
		fmt.Println("⚠️  No standard Blender installation directories detected automatically.")
		fmt.Println("   You can specify your Blender addons directory manually using:")
		fmt.Println("   brc install-addon --path \"<path_to_blender>/scripts/addons\"")
		return nil
	}

	// 4. Select target versions
	var selectedTargets []BlenderTarget

	if allVersions {
		selectedTargets = targets
	} else {
		fmt.Println("Detected Blender installations:")
		for i, t := range targets {
			status := "[Not Installed]"
			addonDir := filepath.Join(t.AddonDir, AddonFolderName)
			if _, err := os.Stat(addonDir); err == nil {
				status = "[Installed]    "
			}
			fmt.Printf("  [%d] %s  %s  -> %s\n", i+1, status, t.VersionName, t.AddonDir)
		}
		fmt.Println("  [a] All versions")
		fmt.Println("  [q] Cancel")
		fmt.Println()

		fmt.Print("Select target version(s) to install (space-separated, e.g. '1 2' or 'a'): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil && len(input) == 0 {
			// Non-interactive fallback (e.g. pipe)
			selectedTargets = targets
		} else {
			input = strings.TrimSpace(input)
			if strings.EqualFold(input, "q") || strings.EqualFold(input, "quit") || strings.EqualFold(input, "exit") {
				fmt.Println("Installation cancelled.")
				return nil
			}
			if input == "" || strings.EqualFold(input, "a") || strings.EqualFold(input, "all") {
				selectedTargets = targets
			} else {
				tokens := strings.Fields(input)
				seen := make(map[int]bool)
				for _, tok := range tokens {
					idx, err := strconv.Atoi(tok)
					if err != nil || idx < 1 || idx > len(targets) {
						return fmt.Errorf("❌ Invalid selection '%s'. Please specify numbers between 1 and %d.", tok, len(targets))
					}
					if !seen[idx] {
						seen[idx] = true
						selectedTargets = append(selectedTargets, targets[idx-1])
					}
				}
			}
		}
	}

	if len(selectedTargets) == 0 {
		fmt.Println("No targets selected.")
		return nil
	}

	// 5. Extract to selected targets
	fmt.Println()
	successCount := 0
	for _, target := range selectedTargets {
		installedDir, err := extractZipFile(zipPath, target.AddonDir)
		if err != nil {
			fmt.Printf("❌ Failed to install to %s (%s): %v\n", target.VersionName, target.AddonDir, err)
		} else {
			fmt.Printf("✓ Installed to %s: %s\n", target.VersionName, installedDir)
			successCount++
		}
	}

	if successCount > 0 {
		printNextSteps()
	}
	return nil
}

func printNextSteps() {
	fmt.Println("\n🎉 Installation complete!")
	fmt.Println("👉 Next steps:")
	fmt.Println("   1. Open Blender")
	fmt.Println("   2. Go to Edit -> Preferences -> Add-ons")
	fmt.Println("   3. Search for 'Blender Remote Console' and check the box to enable it")
	fmt.Println("   4. Press 'N' in 3D Viewport to find the 'Remote Console' tab")
}

func runUninstallAddon(customPath string, allVersions bool) error {

	fmt.Println(">>> Removing Blender Remote Console Addon...")

	if customPath != "" {
		targetAddonDir := filepath.Join(customPath, AddonFolderName)
		if _, err := os.Stat(targetAddonDir); err == nil {
			if err := os.RemoveAll(targetAddonDir); err != nil {
				fmt.Printf("❌ Failed to remove %s: %v\n", targetAddonDir, err)
			} else {
				fmt.Printf("✓ Removed: %s\n", targetAddonDir)
			}
		} else {
			fmt.Printf("  (Not installed in %s)\n", customPath)
		}
		return nil
	}

	targets := detectBlenderTargets()
	if len(targets) == 0 {
		fmt.Println("No Blender installations found.")
		return nil
	}

	var selectedTargets []BlenderTarget
	if allVersions {
		selectedTargets = targets
	} else {
		fmt.Println("Installed Blender versions:")
		for i, t := range targets {
			addonDir := filepath.Join(t.AddonDir, AddonFolderName)
			status := "[Not Installed]"
			if _, err := os.Stat(addonDir); err == nil {
				status = "[Installed]    "
			}
			fmt.Printf("  [%d] %s  %s  -> %s\n", i+1, status, t.VersionName, t.AddonDir)
		}
		fmt.Println("  [a] All versions")
		fmt.Println("  [q] Cancel")
		fmt.Println()

		fmt.Print("Select target version(s) to uninstall (space-separated, e.g. '1 2' or 'a'): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil && len(input) == 0 {
			selectedTargets = targets
		} else {
			input = strings.TrimSpace(input)
			if strings.EqualFold(input, "q") || strings.EqualFold(input, "quit") || strings.EqualFold(input, "exit") {
				fmt.Println("Uninstallation cancelled.")
				return nil
			}
			if input == "" || strings.EqualFold(input, "a") || strings.EqualFold(input, "all") {
				selectedTargets = targets
			} else {
				tokens := strings.Fields(input)
				seen := make(map[int]bool)
				for _, tok := range tokens {
					idx, err := strconv.Atoi(tok)
					if err != nil || idx < 1 || idx > len(targets) {
						return fmt.Errorf("❌ Invalid selection '%s'. Please specify numbers between 1 and %d.", tok, len(targets))
					}
					if !seen[idx] {
						seen[idx] = true
						selectedTargets = append(selectedTargets, targets[idx-1])
					}
				}
			}
		}
	}

	for _, target := range selectedTargets {
		targetAddonDir := filepath.Join(target.AddonDir, AddonFolderName)
		if _, err := os.Stat(targetAddonDir); err == nil {
			if err := os.RemoveAll(targetAddonDir); err != nil {
				fmt.Printf("❌ Failed to remove from %s: %v\n", target.VersionName, err)
			} else {
				fmt.Printf("✓ Removed from %s: %s\n", target.VersionName, targetAddonDir)
			}
		} else {
			fmt.Printf("  (Not installed in %s)\n", target.VersionName)
		}
	}
	fmt.Println("✓ Uninstallation complete.")
	return nil
}

func runDoctor() error {
	fmt.Println("=== Blender Remote Console Doctor ===")
	fmt.Println()

	// 1. Check CLI binary path
	exe, err := os.Executable()
	if err == nil {
		fmt.Printf("✓ CLI Binary: %s\n", exe)
	} else {
		fmt.Printf("❌ Unable to determine CLI binary path: %v\n", err)
	}

	// 2. Check local zip file
	zipPath, err := findLocalZip("")
	if err == nil {
		fmt.Printf("✓ Addon Zip Package: %s\n", zipPath)
	} else {
		fmt.Printf("⚪ Addon Zip Package: Not found in standard paths\n")
	}

	// 3. Check Daemon status
	conn, err := net.DialTimeout("tcp", DaemonAddr, 200*time.Millisecond)
	if err == nil {
		conn.Close()
		fmt.Printf("✓ Daemon: Running (Listening on %s)\n", DaemonAddr)
	} else {
		fmt.Printf("ℹ️ Daemon: Not running (will start automatically on first `brc exec` or `brc sessions`)\n")
	}

	// 4. Check Blender installations & Addon status
	fmt.Println("\nBlender Installations & Addon Status:")
	targets := detectBlenderTargets()
	if len(targets) == 0 {
		fmt.Println("  ⚠️  No standard Blender config directories found.")
	} else {
		for _, t := range targets {
			addonPath := filepath.Join(t.AddonDir, AddonFolderName)
			if _, err := os.Stat(addonPath); err == nil {
				fmt.Printf("  ✓ [Installed]     %s -> %s\n", t.VersionName, addonPath)
			} else {
				fmt.Printf("  ⚪ [Not Installed] %s -> %s\n", t.VersionName, addonPath)
			}
		}
	}

	fmt.Println()
	fmt.Println("Run 'brc install-addon' to install the addon.")
	return nil
}

var installCmd = &cobra.Command{
	Use:   "install-addon",
	Short: "Interactively select & install Blender addon",
	RunE: func(cmd *cobra.Command, args []string) error {
		customPath, _ := cmd.Flags().GetString("path")
		customZip, _ := cmd.Flags().GetString("zip")
		allVersions, _ := cmd.Flags().GetBool("all")
		return runInstallAddon(customPath, customZip, allVersions)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall-addon",
	Short: "Remove Blender addon from versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		customPath, _ := cmd.Flags().GetString("path")
		allVersions, _ := cmd.Flags().GetBool("all")
		return runUninstallAddon(customPath, allVersions)
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system and addon installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(doctorCmd)
	
	// Add flags so they show up in help, even if runInstallAddon parses them via FlagSet
	installCmd.Flags().BoolP("all", "y", false, "Install to all detected Blender versions")
	installCmd.Flags().String("path", "", "Custom path")
	installCmd.Flags().String("zip", "", "Path to zip")
	uninstallCmd.Flags().BoolP("all", "y", false, "Uninstall from all detected Blender versions")
	uninstallCmd.Flags().String("path", "", "Custom path")
}
