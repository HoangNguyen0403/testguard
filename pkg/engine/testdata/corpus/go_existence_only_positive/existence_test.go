package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUser_ExistenceOnly(t *testing.T) {
	user, err := userService.GetUser(1)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}
