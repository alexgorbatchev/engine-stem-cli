package testfixture_test

import (
	"bytes"
	"testing"

	"github.com/alexgorbatchev/engine-stem-cli/internal/crypto"
	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
)

func TestSyntheticBinarySuppliesKey(t *testing.T) {
	want := []byte("synthetic-key123")
	t.Setenv("ENGINE_DJ_APP_PATH", testfixture.EngineBinary(t, want))
	got, err := crypto.DeriveKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("fixture failed to supply its own synthetic key")
	}
}
