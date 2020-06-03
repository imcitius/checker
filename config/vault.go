package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/vault/api"
	"time"
)

func GetVaultSecret(path, field string) (string, error) {

	client, err := api.NewClient(&api.Config{
		Address: Viper.GetString("VAULT_ADDR"),
		Timeout: time.Duration(2 * time.Second),
	})
	if err != nil {
		Log.Infof("Failed to create Vault client: %v", err)
		return "", errors.New(fmt.Sprintf("Failed to create Vault client: %v", err))
	}

	client.SetToken(Viper.GetString("VAULT_TOKEN"))

	sec, err := client.Logical().Read(path)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to get secret: %v", err))
	}
	if sec == nil || sec.Data == nil {
		return "", errors.New(fmt.Sprintf("No data for key %s\n", field))
	}
	return fmt.Sprint(sec.Data[field]), nil
}
