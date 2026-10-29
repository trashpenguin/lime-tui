package main

import (
	"bytes"
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
	"unsafe"
)

var secretPurpose = []byte("Lime TUI optional account credential v1")

func protectProfileSecret(data []byte, decrypt bool) ([]byte, error) {
	if len(data) == 0 || len(data) > 65536 {
		return nil, errors.New("invalid credential size")
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	entropy := windows.DataBlob{Size: uint32(len(secretPurpose)), Data: &secretPurpose[0]}
	var output windows.DataBlob
	var err error
	if decrypt {
		err = windows.CryptUnprotectData(&input, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output)
	} else {
		err = windows.CryptProtectData(&input, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output)
	}
	if err != nil {
		return nil, err
	}
	if output.Data == nil {
		return nil, errors.New("DPAPI returned no credential")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	raw := unsafe.Slice(output.Data, int(output.Size))
	result := bytes.Clone(raw)
	clear(raw)
	return result, nil
}

func boundedSecret(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, errors.New("invalid credential file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil {
		return nil, err
	}
	if len(data) > 65536 {
		return nil, errors.New("credential file too large")
	}
	return data, nil
}

func writeProfileSecret(path, value string) error {
	data := []byte(value)
	defer clear(data)
	protected, err := protectProfileSecret(data, false)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".lime-secret-*.tmp")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(protected); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	target := path + ".dpapi"
	if err = os.Rename(temp, target); err != nil {
		return err
	}
	roundtrip, err := readProtectedSecret(target)
	if err != nil {
		return err
	}
	if roundtrip != value {
		return errors.New("credential verification failed")
	}
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func readProtectedSecret(path string) (string, error) {
	data, err := boundedSecret(path)
	if err != nil {
		return "", err
	}
	plain, err := protectProfileSecret(data, true)
	if err != nil {
		return "", err
	}
	defer clear(plain)
	return string(plain), nil
}

func readProfileSecret(path string) (string, error) {
	value, err := readProtectedSecret(path + ".dpapi")
	if err == nil {
		if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
			return "", e
		}
		return value, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	data, err := boundedSecret(path)
	if err != nil {
		return "", err
	}
	defer clear(data)
	value = string(data)
	if err = writeProfileSecret(path, value); err != nil {
		return "", err
	}
	return value, nil
}
