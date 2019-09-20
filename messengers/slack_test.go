package messengers_test

import (
	"testing"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/messengers"
	"github.com/stretchr/testify/assert"
)

func TestSlackValidateWhenInvalid(t *testing.T) {
	s := messengers.NewSlack("token", "")
	assert.Equal(t, false, s.Validate(), "wrong validate")
}

func TestSlackValidateWhenValid(t *testing.T) {
	s := messengers.NewSlack("token", "channel")
	assert.Equal(t, true, s.Validate(), "wrong validate")
}
