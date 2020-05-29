package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	"time"
)

func GetVaultSecret(path, field string) (string, error) {
	var (
		vaultToken, vaultAddress string
	)
	vaultToken = viper.GetString("VAULT_TOKEN")
	vaultAddress = viper.GetString("VAULT_ADDR")

	client, err := api.NewClient(&api.Config{
		Address: vaultAddress,
		Timeout: time.Duration(2 * time.Second),
	})
	if err != nil {
		Log.Infof("Failed to create Vault client: %v", err)
		return "", errors.New(fmt.Sprintf("Failed to create Vault client: %v", err))
	}

	client.SetToken(vaultToken)

	sec, err := client.Logical().Read(path)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to get secret: %v", err))
	}
	if sec == nil || sec.Data == nil {
		return "", errors.New(fmt.Sprintf("No data for key %s\n", field))
	}
	return fmt.Sprint(sec.Data[field]), nil
}
