//go:build darwin

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDarwinKeychainResolutionDoesNotGuessFromHome(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom login.keychain-db")
	if err := os.WriteFile(path, []byte("test keychain"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		name, output string
		commandError error
		valid        bool
	}{
		{name: "quoted path", output: "    \"" + path + "\"\n", valid: true},
		{name: "command failed", output: "A default keychain could not be found.", commandError: errors.New("exit status 1")},
		{name: "empty output"},
		{name: "relative path", output: "login.keychain-db"},
		{name: "missing file", output: filepath.Join(dir, "missing.keychain-db")},
		{name: "directory", output: dir},
		{name: "multiple lines", output: path + "\nother"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDarwinUserKeychain(func(name string, args ...string) (string, error) {
				if name != "security" || !reflect.DeepEqual(args, []string{"default-keychain", "-d", "user"}) {
					t.Fatalf("unexpected keychain command: %s %v", name, args)
				}
				return tc.output, tc.commandError
			}, os.Stat)
			if tc.valid {
				if err != nil || got != path {
					t.Fatalf("got %q, %v", got, err)
				}
			} else if got != "" || !errors.Is(err, ErrUserKeychainUnavailable) {
				t.Fatalf("invalid keychain did not fail closed: %q, %v", got, err)
			}
		})
	}
}

func TestDarwinCertInstallStopsBeforeMutationWhenKeychainUnavailable(t *testing.T) {
	err := darwinInstallCert("/preview/ca.crt", func() (string, error) {
		return "", ErrUserKeychainUnavailable
	}, func(name string, args ...string) (string, error) {
		t.Fatalf("certificate command ran without a keychain: %s %v", name, args)
		return "", nil
	})
	if !errors.Is(err, ErrUserKeychainUnavailable) {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestDarwinCertInstallUsesOnlyResolvedUserKeychain(t *testing.T) {
	for _, failAdd := range []bool{false, true} {
		var commands [][]string
		err := darwinInstallCert("/preview/ca.crt", func() (string, error) {
			return "/real-user/custom.keychain-db", nil
		}, func(name string, args ...string) (string, error) {
			commands = append(commands, append([]string{name}, args...))
			if failAdd {
				return "denied", errors.New("exit status 1")
			}
			return "", nil
		})
		want := [][]string{
			{"security", "add-certificates", "-k", "/real-user/custom.keychain-db", "/preview/ca.crt"},
			{"security", "add-trusted-cert", "-r", "trustRoot", "-k", "/real-user/custom.keychain-db", "/preview/ca.crt"},
		}
		if failAdd {
			want = want[:1]
		}
		if (err != nil) != failAdd || !reflect.DeepEqual(commands, want) {
			t.Fatalf("failAdd=%v: commands=%v, error=%v", failAdd, commands, err)
		}
	}
}

func TestDarwinSetSystemProxyCommandsUsePACWhenWebPortAvailable(t *testing.T) {
	commands := darwinSetSystemProxyCommands([]string{"Wi-Fi"}, 8080, 52123)
	want := [][]string{
		{"networksetup", "-setautoproxyurl", "Wi-Fi", "http://127.0.0.1:52123/proxy.pac?proxy=8080"},
		{"networksetup", "-setautoproxystate", "Wi-Fi", "on"},
		{"networksetup", "-setwebproxystate", "Wi-Fi", "off"},
		{"networksetup", "-setsecurewebproxystate", "Wi-Fi", "off"},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", want, commands)
	}
}

func TestDarwinSetSystemProxyCommandsDisablePACBeforeManualProxy(t *testing.T) {
	commands := darwinSetSystemProxyCommands([]string{"Wi-Fi"}, 8080, 0)
	want := [][]string{
		{"networksetup", "-setautoproxystate", "Wi-Fi", "off"},
		{"networksetup", "-setwebproxy", "Wi-Fi", "127.0.0.1", "8080"},
		{"networksetup", "-setsecurewebproxy", "Wi-Fi", "127.0.0.1", "8080"},
		{"networksetup", "-setwebproxystate", "Wi-Fi", "on"},
		{"networksetup", "-setsecurewebproxystate", "Wi-Fi", "on"},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", want, commands)
	}
}

func TestDarwinClearSystemProxyCommandsDisablePACAndManualProxy(t *testing.T) {
	commands := darwinClearSystemProxyCommands([]string{"Wi-Fi"})
	want := [][]string{
		{"networksetup", "-setautoproxystate", "Wi-Fi", "off"},
		{"networksetup", "-setwebproxystate", "Wi-Fi", "off"},
		{"networksetup", "-setsecurewebproxystate", "Wi-Fi", "off"},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", want, commands)
	}
}

func TestDarwinRunSystemProxyCommandsContinuesAndReturnsFailures(t *testing.T) {
	commands := [][]string{
		{"networksetup", "-setautoproxyurl", "Wi-Fi", "http://127.0.0.1:52123/proxy.pac?proxy=8080"},
		{"networksetup", "-setautoproxystate", "Wi-Fi", "on"},
		{"networksetup", "-setwebproxystate", "Wi-Fi", "off"},
	}
	var gotCommands [][]string

	err := darwinRunSystemProxyCommands(commands, func(name string, args ...string) (string, error) {
		command := append([]string{name}, args...)
		gotCommands = append(gotCommands, command)
		switch len(gotCommands) {
		case 1:
			return "PAC failed\n", errors.New("exit status 4")
		case 3:
			return "web proxy failed\n", errors.New("exit status 5")
		default:
			return "", nil
		}
	})

	if !reflect.DeepEqual(gotCommands, commands) {
		t.Fatalf("commands executed mismatch\nwant: %#v\n got: %#v", commands, gotCommands)
	}
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	msg := err.Error()
	for _, want := range []string{
		"networksetup -setautoproxyurl Wi-Fi http://127.0.0.1:52123/proxy.pac?proxy=8080",
		"PAC failed",
		"networksetup -setwebproxystate Wi-Fi off",
		"web proxy failed",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
}
