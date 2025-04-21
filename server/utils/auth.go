package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// This package provides utility functions for authentication and authorization.
// It includes functions for simple password hashing and verification

// HashPassword hashes a password using a password hashing algorithm.
// We will use the default cost for now.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(hashedPassword))
	return err == nil
}

func CompareHash(hash1, hash2 string) bool {
	return hash1 == hash2
}
