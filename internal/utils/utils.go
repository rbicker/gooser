package utils

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"time"
)

// init function is used to seed random.
func init() {
	rand.Seed(time.Now().UnixNano())
}

// LookupEnv loads the setting with the given name
// from the environment. If no environment variable
// with the given name is set, the defaultValue is
// returned.
func LookupEnv(name, defaultValue string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return defaultValue
}

// IsMailAddress checks if the given string is a mail address.
func IsMailAddress(s string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(s)
}

// RandomString is used to generate a random string with n letters.
func RandomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RemoveFromStringSlice removes the first occurrence of the
// given element from the slice of strings.
//
func RemoveFromStringSlice(ss []string, s string) []string {
	for i, x := range ss {
		if s == x {
			ss[i] = ss[len(ss)-1]
			return ss[:len(ss)-1]
		}
	}
	return ss
}

// AppendUniqueString appends the given string to the given slice
// if the string does not already exist in the slice.
// A copy of the resulting slice and a bool which is set to true
// if slice has changed are being returned.
func AppendUniqueString(ss []string, s string) ([]string, bool) {
	for _, v := range ss {
		if s == v {
			// given string is already in slice
			return ss, false
		}
	}
	return append(ss, s), true
}

// UniqueStringSlice makes sure all the strings in the given slice are unique.
// A copy of the resulting slice and a bool which is set to true
// if slice has changed are being returned.
func UniqueStringSlice(ss []string) ([]string, bool) {
	var res []string
	for _, s := range ss {
		res, _ = AppendUniqueString(res, s)
	}
	return res, len(ss) != len(res)
}

// StringSlicesDiff compares to string slices and returns all the strings
// that were added to the second slice and all the strings that were removed
// from the first slice.
// The slices need to be unique otherwise the function might return
// unexpected results.
func StringSlicesDiff(a, b []string) (added []string, removed []string) {
	// maps to enable looking up values
	ma := make(map[string]struct{}, len(a))
	mb := make(map[string]struct{}, len(b))
	// loop over first slice
	// add a empty struct (does not use memory)
	// for all values to the map
	for _, x := range a {
		ma[x] = struct{}{}
	}
	// loop over second slice
	for _, x := range b {
		// if value is not in first slice
		// it was added
		if _, found := ma[x]; !found {
			added = append(added, x)
		} else {
			// if it was not added
			// store it in the second map
			// which will be used to check
			// which values were removed
			mb[x] = struct{}{}
		}
	}
	// loop over first slice again
	// to check which values were removed
	for _, x := range a {
		//
		if _, found := mb[x]; !found {
			removed = append(removed, x)
		}
	}
	return added, removed
}

// Encrypt encrypts the given message using the given key.
func Encrypt(key, msg string) (string, error) {
	plain := []byte(msg)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	// iv needs to be unique, but doesn't have to be secure.
	// it's common to put it at the beginning of the cipher text.
	cypher := make([]byte, aes.BlockSize+len(plain))
	iv := cypher[:aes.BlockSize]
	if _, err = io.ReadFull(crand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cypher[aes.BlockSize:], plain)
	// return base64 encoded string
	res := base64.URLEncoding.EncodeToString(cypher)
	return res, nil
}

// Decrypt decrypts the given message the given key.
func Decrypt(key, msg string) (string, error) {
	cipherText, err := base64.URLEncoding.DecodeString(msg)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("cipher text block size is to short")
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)
	return string(cipherText), nil
}
