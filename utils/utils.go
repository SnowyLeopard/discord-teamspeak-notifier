package utils

import (
	"math/rand"
	"time"
)

type Set map[string]struct{}

// Adds an  to the set
func (s Set) Add(value string) {
	s[value] = struct{}{}
}

// Removes an  from the set
func (s Set) Remove(value string) {
	delete(s, value)
}

func (s Set) Has(value string) bool {
	_, ok := s[value]
	return ok
}


func RandomString() string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321"
	b := make([]byte, 20)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset)-1)]
	}
	return string(b)
}