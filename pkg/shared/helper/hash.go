package helper

import (
	"crypto/md5"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// CheckPasswordHash - Method to compare password hash
func CheckPasswordHash(password string, hash primitive.Binary) bool {
	err := bcrypt.CompareHashAndPassword(hash.Data, []byte(password))
	return err == nil
}

// GeneratePasswordHash - Method to generate password hash
func GeneratePasswordHash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func PasswordHash(password string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(password)))
}

func ValidatePassword(password string, hashstring string) bool {
	if PasswordHash(password) == hashstring {
		return true
	}
	return false
}
