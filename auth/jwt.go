package auth

import (
	"fmt"
	"github.com/cristalhq/jwt/v3"
	"my/checker/config"
)

var (
	key = &config.Config.Defaults.TokenEncryptionKey
)

func GenerateToken() {

	err := config.LoadConfig()
	if err != nil {
		config.Log.Infof("Config load error: %s", err)
	}

	signer, err := jwt.NewSignerHS(jwt.HS256, *key)
	if err != nil {
		config.Log.Infof("Cannot generate token signer: %s", err.Error())
		return
	}

	claims := &jwt.StandardClaims{
		Audience: []string{"admin"},
		ID:       "Oi3ooxie4aikeimoozo8Egai6aiz9poh",
	}

	// 3. create a builder
	builder := jwt.NewBuilder(signer)

	token, err := builder.Build(claims)
	if err != nil {
		config.Log.Infof("Cannot generate token builder: %s", err.Error())
		return
	}

	fmt.Printf("Jwt token: %s\n", token)
}

func checkErr(err error) {
	if err != nil {
		return
	}
}
