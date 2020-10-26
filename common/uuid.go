package common

import (
	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	"math/rand"
)

func GetRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func GenUUID(host string) string {
	var err error

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		return ""
	}

	u2 := uuid.NewSHA1(ns, []byte(host))
	return u2.String()
}
