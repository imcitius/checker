package auth

import (
	"fmt"
	"github.com/cristalhq/jwt/v3"
	"my/checker/config"
)

var key []byte

func GenerateToken() {

	err := config.LoadConfig()
	if err != nil {
		config.Log.Errorf("Config load error: %s", err)
	}

	if config.Koanf.String("checker.token.encryption.key") != "" {
		key = config.Koanf.Bytes("checker.token.encryption.key")
	} else {
		key = config.TokenEncryptionKey
	}

	signer, err := jwt.NewSignerHS(jwt.HS256, key)
	if err != nil {
		config.Log.Errorf("Cannot generate token signer: %s", err.Error())
		return
	}

	claims := &jwt.StandardClaims{
		Audience: []string{"admin"},
		ID:       "Oi3ooxie4aikeimoozo8Egai6aiz9poh",
	}

	builder := jwt.NewBuilder(signer)

	token, err := builder.Build(claims)
	if err != nil {
		config.Log.Errorf("Cannot generate token builder: %s", err.Error())
		return
	}

	fmt.Printf("Jwt token: %s\n", token)
}
