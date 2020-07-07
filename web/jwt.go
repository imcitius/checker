package web

import (
	"github.com/cristalhq/jwt/v3"
	"my/checker/config"
)

var (
	key = []byte(`bi6oNuisa0ooz6Ael6Eewaatoophoo0p`)
)

func GenerateToken() {
	signer, err := jwt.NewSignerHS(jwt.HS256, key)
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

	config.Log.Infof("Jwt token: %s", token)
}

func checkErr(err error) {
	if err != nil {
		return
	}
}
