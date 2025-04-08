package persistence

import (
	"os"
	"path/filepath"
)

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./.video_compressor_config"
	}
	return filepath.Join(homeDir, ".video_compressor_config")
}

func SaveLastUsedFolder(path string) {
	configPath := getConfigPath()
	folder := filepath.Dir(path)
	os.WriteFile(configPath, []byte(folder), 0644)
}

func LoadLastUsedFolder() string {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "" // fallback to default if not found
	}
	return string(data)
}
