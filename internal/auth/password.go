package auth

import (
	"crypto/rand"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	tempPassLength = 12
)

// CompareHashAndPassword...
func CompareHashAndPassword(hashed string, plain []byte) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), plain)
	return err == nil
}

// GeneratePassword generates password considering alphanumerics and selected symbols
func GeneratePassword() (*string, error) {
	var charsetBuilder strings.Builder
	charsetBuilder.WriteString("#$!-_")
	for ch := 'a'; ch <= 'z'; ch++ {
		charsetBuilder.WriteRune(ch)
	}
	for ch := 'A'; ch <= 'Z'; ch++ {
		charsetBuilder.WriteRune(ch)
	}
	for ch := '0'; ch <= '9'; ch++ {
		charsetBuilder.WriteRune(ch)
	}

	charset := charsetBuilder.String()
	password := make([]byte, tempPassLength)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(charsetBuilder.Len())))
		if err != nil {
			return nil, err
		}
		password[i] = charset[num.Int64()]
	}
	generatedPass := string(password)
	return &generatedPass, nil
}

// GeneratePasswordHash...
func GeneratePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
