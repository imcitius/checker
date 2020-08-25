package config

import (
	"fmt"
	vaultApi "github.com/hashicorp/vault/api"
	"strings"
	"time"
)

var VaultClient vaultApi.Client

func init() {
	Secrets = make(map[string]CachedSecret)

	//Log.Debugf("GetVaultSecret: vaultPath=%s", vaultPath)

	VaultClient, err := vaultApi.NewClient(&vaultApi.Config{
		Address: Koanf.String("vault.addr"),
		Timeout: 3 * time.Second,
	})
	if err != nil {
		Log.Warnf("failed to create Vault client: %v", err)
	}

	VaultClient.SetToken(Koanf.String("vault.token"))

}

func GetVaultSecret(vaultPath string) (string, error) {

	vault := strings.Split(vaultPath, ":")
	path := vault[1]
	field := vault[2]
	if path == "" {
		return "", fmt.Errorf("failed to get secret, vault path is empty")
	}
	if field == "" {
		return "", fmt.Errorf("failed to get secret, field name is empty")
	}

	if token, ok := Secrets[path]; ok {
		if time.Now().Sub(token.TimeStamp) < 5*time.Minute {
			//Log.Debugf("Secret return from cache")
			return token.Secret, nil
		}
	}

	sec, err := VaultClient.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %v", err)
	}
	if sec == nil || sec.Data == nil {
		return "", fmt.Errorf("no data for key %s", field)
	}
	Secrets[path] = CachedSecret{fmt.Sprint(sec.Data[field]), time.Now()}

	return fmt.Sprint(sec.Data[field]), nil
}
