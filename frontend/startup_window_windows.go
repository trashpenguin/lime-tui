package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Explorer launches get a dedicated, comfortably sized terminal. Invocations
// from an existing shell keep that shell's window unless --window is requested.
func launchDefaultWindow() bool {
	if len(os.Args) == 2 && os.Args[1] == "--terminal-window" {
		os.Args = os.Args[:1]
		return false
	}
	explicit := len(os.Args) == 2 && os.Args[1] == "--window"
	if !explicit && (len(os.Args) != 1 || !explorerParent()) {
		return false
	}
	if explicit {
		os.Args = os.Args[:1]
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	terminal, err := exec.LookPath("wt.exe")
	if err != nil {
		return false // The existing console still works without Windows Terminal.
	}
	cmd := exec.Command(terminal, "-w", "new", "--size", "152,38", "-d", filepath.Dir(exe), exe, "--terminal-window")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd.Run() == nil
}

func explorerParent() bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if entry.ProcessID == uint32(os.Getppid()) {
			return strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), "explorer.exe")
		}
	}
	return false
}
