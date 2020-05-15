package main

import (
	"testing"
)

var ConfigTest ConfigFile

func TestJsonLoad(t *testing.T) {
	err := jsonLoad("config.json", ConfigTest)
	if err != nil {
		t.Error("Test config load error: ", err)
	}
}
