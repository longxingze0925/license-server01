package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := "test@test.com"
	password := "Test123456"

	// SDK 预哈希方式: SHA256(password + ":" + email + ":license_salt_v1")
	salted := fmt.Sprintf("%s:%s:license_salt_v1", password, strings.ToLower(email))
	hash := sha256.Sum256([]byte(salted))
	preHashed := hex.EncodeToString(hash[:])

	fmt.Printf("原始密码: %s\n", password)
	fmt.Printf("预哈希后: %s\n", preHashed)

	// 生成 bcrypt 哈希
	bcryptHash, _ := bcrypt.GenerateFromPassword([]byte(preHashed), bcrypt.DefaultCost)
	fmt.Printf("bcrypt哈希: %s\n", string(bcryptHash))
}
