package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUser_PayloadChecked(t *testing.T) {
	user, err := userService.GetUser(1)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Alice", user.Name)
}

func TestGetUser_ErrorCase(t *testing.T) {
	user, err := userService.GetUser(-1)
	assert.Error(t, err)
	assert.Nil(t, user)
}
