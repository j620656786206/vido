// Package crypto provides key derivation functions for encryption operations.
package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// KeySource represents the source of the encryption key.
type KeySource string

const (
	// KeySourceEnvVar indicates the key was derived from ENCRYPTION_KEY environment variable.
	KeySourceEnvVar KeySource = "env_var"

	// KeySourceMachineID indicates the key was derived from machine ID (fallback).
	KeySourceMachineID KeySource = "machine_id"

	// pbkdf2Iterations is the number of PBKDF2 iterations for key derivation.
	pbkdf2Iterations = 100000

	// saltDefault is the default salt for key derivation.
	// Using a fixed salt is acceptable since the goal is deterministic key derivation.
	saltDefault = "vido-secrets-salt-v1"
)

var (
	// ErrEncryptionKeyNotSet indicates ENCRYPTION_KEY environment variable is not set.
	ErrEncryptionKeyNotSet = errors.New("ENCRYPTION_KEY environment variable not set")

	// ErrMachineIDNotFound indicates machine ID could not be determined.
	ErrMachineIDNotFound = errors.New("machine ID not found")
)

// DeriveKey returns the encryption key, preferring env var over machine ID.
// Returns the derived key, the source of the key, and any error.
func DeriveKey() ([]byte, KeySource, error) {
	// Try environment variable first
	key, err := DeriveKeyFromEnv()
	if err == nil {
		slog.Info("Using encryption key from ENCRYPTION_KEY environment variable")
		return key, KeySourceEnvVar, nil
	}

	// Fallback to machine ID
	slog.Warn("ENCRYPTION_KEY not set, using machine ID as fallback",
		"recommendation", "Set ENCRYPTION_KEY environment variable for better security")

	key, err = DeriveKeyFromMachineID()
	if err != nil {
		return nil, "", err
	}

	return key, KeySourceMachineID, nil
}

// DeriveKeyFromEnv reads the ENCRYPTION_KEY environment variable and derives a 32-byte key.
func DeriveKeyFromEnv() ([]byte, error) {
	envKey := os.Getenv("ENCRYPTION_KEY")
	if envKey == "" {
		return nil, ErrEncryptionKeyNotSet
	}

	return deriveKeyFromString(envKey), nil
}

// DeriveKeyFromMachineID derives a key from the machine's unique identifier.
// This is a fallback when ENCRYPTION_KEY is not set.
func DeriveKeyFromMachineID() ([]byte, error) {
	machineID, err := getMachineID()
	if err != nil {
		return nil, err
	}

	return deriveKeyFromString(machineID), nil
}

// deriveKeyFromString derives a 32-byte key from an input string using PBKDF2.
func deriveKeyFromString(input string) []byte {
	return pbkdf2.Key([]byte(input), []byte(saltDefault), pbkdf2Iterations, KeySize, sha256.New)
}

// getMachineID returns a unique identifier for the current machine.
// Supports Linux, macOS, and Windows.
func getMachineID() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxMachineID()
	case "darwin":
		return getDarwinMachineID()
	case "windows":
		return getWindowsMachineID()
	default:
		return getFallbackMachineID()
	}
}

// getLinuxMachineID reads machine ID from /etc/machine-id or /var/lib/dbus/machine-id.
func getLinuxMachineID() (string, error) {
	// Try /etc/machine-id first
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	}

	// Fallback to /var/lib/dbus/machine-id
	if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	}

	return getFallbackMachineID()
}

// getDarwinMachineID reads IOPlatformUUID on macOS.
func getDarwinMachineID() (string, error) {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return getFallbackMachineID()
	}

	// Parse IOPlatformUUID from output
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				uuid := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				if uuid != "" {
					return uuid, nil
				}
			}
		}
	}

	return getFallbackMachineID()
}

// getWindowsMachineID reads MachineGuid from Windows registry.
func getWindowsMachineID() (string, error) {
	out, err := exec.Command("reg", "query",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography",
		"/v", "MachineGuid").Output()
	if err != nil {
		return getFallbackMachineID()
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MachineGuid") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[len(fields)-1], nil
			}
		}
	}

	return getFallbackMachineID()
}

// getFallbackMachineID generates a fallback identifier based on hostname.
// This is used when platform-specific machine ID is not available.
func getFallbackMachineID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	hash := sha256.Sum256([]byte(hostname + "vido-fallback"))
	return hex.EncodeToString(hash[:]), nil
}
