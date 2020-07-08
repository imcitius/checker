package config

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"strings"
	"time"
)

func init() {
	ClearSecrets()
}

func ClearSecrets() {
	// Secret cache, to reduce Vault requests number
	Secrets = make(map[string]string)
}

func GetVaultSecret(vaultPath string) (string, error) {

	//Log.Debugf("GetVaultSecret: vaultPath=%s", vaultPath)
	vault := strings.Split(vaultPath, ":")
	path := vault[1]
	field := vault[2]

	if token, ok := Secrets[path]; ok {
		//Log.Debugf("Secret return from cache")
		return token, nil
	}

	client, err := api.NewClient(&api.Config{
		Address: Koanf.String("vault.addr"),
		Timeout: time.Duration(3 * time.Second),
	})
	if err != nil {
		Log.Infof("Failed to create Vault client: %v", err)
		return "", fmt.Errorf("failed to create Vault client: %v", err)
	}

	client.SetToken(Koanf.String("vault.token"))

	sec, err := client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %v", err)
	}
	if sec == nil || sec.Data == nil {
		return "", fmt.Errorf("no data for key %s", field)
	}
	Secrets[path] = fmt.Sprint(sec.Data[field])
	return fmt.Sprint(sec.Data[field]), nil
}
