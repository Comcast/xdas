package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

func main() {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		log.Panicln("unable to generate key", err)
	}
	hexKey := hex.EncodeToString(aesKey)
	fmt.Println(hexKey)
}
