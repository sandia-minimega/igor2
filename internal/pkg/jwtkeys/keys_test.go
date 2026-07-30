// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package jwtkeys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureCreatesMatchingKeyPair(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")

	if err := Ensure(privatePath, publicPath); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	privateKey, err := LoadPrivate(privatePath)
	if err != nil {
		t.Fatalf("LoadPrivate() error = %v", err)
	}
	publicKey, err := LoadPublic(publicPath)
	if err != nil {
		t.Fatalf("LoadPublic() error = %v", err)
	}

	digest := sha256.Sum256([]byte("igor JWT signing test"))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("VerifyPKCS1v15() error = %v", err)
	}

	if runtime.GOOS != "windows" {
		assertMode(t, privatePath, 0600)
		assertMode(t, publicPath, 0644)
		assertMode(t, dir, 0700)
	}
}

func TestEnsureRestoresPublicKey(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")

	if err := Ensure(privatePath, publicPath); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	privateKey, err := LoadPrivate(privatePath)
	if err != nil {
		t.Fatalf("LoadPrivate() error = %v", err)
	}
	if err := os.Remove(publicPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if err := Ensure(privatePath, publicPath); err != nil {
		t.Fatalf("Ensure() restoring public key error = %v", err)
	}
	publicKey, err := LoadPublic(publicPath)
	if err != nil {
		t.Fatalf("LoadPublic() error = %v", err)
	}
	if privateKey.PublicKey.E != publicKey.E || privateKey.PublicKey.N.Cmp(publicKey.N) != 0 {
		t.Fatal("restored public key does not match private key")
	}
}

func TestEnsureRejectsMismatchedPublicKey(t *testing.T) {
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	firstPrivate := filepath.Join(firstDir, "private.pem")
	firstPublic := filepath.Join(firstDir, "public.pem")
	secondPrivate := filepath.Join(secondDir, "private.pem")
	secondPublic := filepath.Join(secondDir, "public.pem")

	if err := Ensure(firstPrivate, firstPublic); err != nil {
		t.Fatalf("Ensure(first) error = %v", err)
	}
	if err := Ensure(secondPrivate, secondPublic); err != nil {
		t.Fatalf("Ensure(second) error = %v", err)
	}
	otherPublic, err := os.ReadFile(secondPublic)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(firstPublic, otherPublic, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Ensure(firstPrivate, firstPublic); err == nil {
		t.Fatal("Ensure() accepted a mismatched public key")
	}
}

func TestDeleteRemovesKeyPair(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")

	if err := Ensure(privatePath, publicPath); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if err := Delete(privatePath, publicPath); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	for _, path := range []string{privatePath, publicPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists after Delete()", path)
		}
	}
	if err := Delete(privatePath, publicPath); err != nil {
		t.Fatalf("Delete() should be idempotent, error = %v", err)
	}
}

func assertMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if actual := info.Mode().Perm(); actual != expected {
		t.Fatalf("mode for %s = %04o, want %04o", path, actual, expected)
	}
}
