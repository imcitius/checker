package config

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"strings"
	"time"
)

func GetVaultSecret(vaultPath string) (string, error) {

	Log.Debugf("GetVaultSecret: vaultPath=%s", vaultPath)
	vault := strings.Split(vaultPath, ":")
	path := vault[1]
	field := vault[2]

	client, err := api.NewClient(&api.Config{
		Address: Viper.GetString("VAULT_ADDR"),
		Timeout: time.Duration(3 * time.Second),
	})
	if err != nil {
		Log.Infof("Failed to create Vault client: %v", err)
		return "", fmt.Errorf("Failed to create Vault client: %v", err)
	}

	client.SetToken(Viper.GetString("VAULT_TOKEN"))

	sec, err := client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("Failed to get secret: %v", err)
	}
	if sec == nil || sec.Data == nil {
		return "", fmt.Errorf("No data for key %s\n", field)
	}
	return fmt.Sprint(sec.Data[field]), nil
}
