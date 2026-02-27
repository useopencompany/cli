package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.agentprotocol.cloud/cli/internal/config"
)

const vaultFile = "vault.json"

// Vault manages encrypted key storage.
// In the full implementation this syncs with the opencompany key vault service.
type Vault struct {
	Keys map[string]string `json:"keys"`
}

// Open loads the vault from disk.
func Open() (*Vault, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, vaultFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Vault{Keys: make(map[string]string)}, nil
	}
	if err != nil {
		return nil, err
	}

	var v Vault
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

// Save persists the vault to disk.
func (v *Vault) Save() error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, vaultFile), data, 0o600)
}

// Set stores a key in the vault.
func (v *Vault) Set(name, value string) error {
	// TODO: Encrypt value before storage.
	// TODO: Sync with opencompany vault API.
	v.Keys[name] = value
	return v.Save()
}

// Get retrieves a key from the vault.
func (v *Vault) Get(name string) (string, error) {
	val, ok := v.Keys[name]
	if !ok {
		return "", fmt.Errorf("key %q not found in vault", name)
	}
	// TODO: Decrypt value.
	return val, nil
}
