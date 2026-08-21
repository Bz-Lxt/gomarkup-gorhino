package validate_test

import (
	"testing"

	"gorhino/internal/shared/validate"
)

func TestCheckURLWhitelistHostIsCaseInsensitive(t *testing.T) {
	err := validate.CheckURL(
		"http://127.0.0.1:8088/echo",
		[]string{"HTTP://127.0.0.1:8088"},
	)
	if err != nil {
		t.Fatalf("equivalent HTTP whitelist address was rejected: %v", err)
	}
}
