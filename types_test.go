package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	userInput := &CreateNewUserInput{
		Name:     "John Doe",
		Email:    "john.doe@gmail.com",
		Password: "test1234",
	}
	user, err := NewUser(*userInput)
	assert.Nil(t, err)
	fmt.Printf("user %v", user)
}
