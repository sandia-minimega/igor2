// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package jwtkeys

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const rsaKeyBits = 3072

// Ensure creates an RSA signing key pair when one is not already present and
// validates that an existing public key belongs to the private key.
func Ensure(privatePath, publicPath string) error {
	privateExists, err := pathExists(privatePath)
	if err != nil {
		return err
	}
	publicExists, err := pathExists(publicPath)
	if err != nil {
		return err
	}

	if !privateExists {
		return generate(privatePath, publicPath)
	}

	privateKey, err := LoadPrivate(privatePath)
	if err != nil {
		return fmt.Errorf("load JWT private key: %w", err)
	}
	if err := os.Chmod(privatePath, 0600); err != nil {
		return fmt.Errorf("protect JWT private key: %w", err)
	}

	if !publicExists {
		if err := writePublic(publicPath, &privateKey.PublicKey); err != nil {
			return fmt.Errorf("restore JWT public key: %w", err)
		}
		return nil
	}

	publicKey, err := LoadPublic(publicPath)
	if err != nil {
		return fmt.Errorf("load JWT public key: %w", err)
	}
	if privateKey.PublicKey.E != publicKey.E || privateKey.PublicKey.N.Cmp(publicKey.N) != 0 {
		return errors.New("JWT public key does not match private key")
	}
	if err := os.Chmod(publicPath, 0644); err != nil {
		return fmt.Errorf("set JWT public key permissions: %w", err)
	}

	return nil
}

// Delete removes both sides of a JWT signing key pair.
func Delete(privatePath, publicPath string) error {
	var errs []error
	for _, path := range []string{privatePath, publicPath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

// LoadPrivate reads an RSA private key from a PKCS#8 or PKCS#1 PEM file.
func LoadPrivate(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, rest := pem.Decode(data)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("invalid private key PEM")
	}

	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
	}

	if err := privateKey.Validate(); err != nil {
		return nil, err
	}
	return privateKey, nil
}

// LoadPublic reads an RSA public key from a PKIX or PKCS#1 PEM file.
func LoadPublic(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, rest := pem.Decode(data)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("invalid public key PEM")
	}

	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return publicKey, nil
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported public key PEM type %q", block.Type)
	}
}

func generate(privatePath, publicPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return fmt.Errorf("generate JWT RSA key: %w", err)
	}
	if err := privateKey.Validate(); err != nil {
		return fmt.Errorf("validate JWT RSA key: %w", err)
	}

	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal JWT private key: %w", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	if err := writeAtomic(privatePath, privatePEM, 0600); err != nil {
		return fmt.Errorf("write JWT private key: %w", err)
	}
	if err := writePublic(publicPath, &privateKey.PublicKey); err != nil {
		return fmt.Errorf("write JWT public key: %w", err)
	}
	return nil
}

func writePublic(path string, publicKey *rsa.PublicKey) error {
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	return writeAtomic(path, publicPEM, 0644)
}

func writeAtomic(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}

	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
