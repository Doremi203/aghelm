package deps

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func stubUID(t *testing.T, uid int) {
	t.Helper()
	original := getUID
	getUID = func() int { return uid }
	t.Cleanup(func() { getUID = original })
}

func stubPath(t *testing.T, present ...string) {
	t.Helper()
	found := make(map[string]bool, len(present))
	for _, name := range present {
		found[name] = true
	}
	original := lookPath
	lookPath = func(name string) (string, error) {
		if found[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = original })
}

func TestInstallCommandPrefersNativeManagerOverBrew(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "brew", "apt-get", "sudo")
	if got, want := installCommand("linux", "tmux"), "sudo apt-get update && sudo apt-get install -y tmux"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestInstallCommandOnDarwinUsesBrew(t *testing.T) {
	stubPath(t, "brew", "apt-get")
	if got, want := installCommand("darwin", "git"), "brew install git"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestHintFallsBackWithoutAKnownManager(t *testing.T) {
	stubPath(t)
	if got, want := hint("linux", "tmux"), "install tmux with your package manager"; got != want {
		t.Fatalf("hint = %q, want %q", got, want)
	}
}

func TestHintNamesTheCommand(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "pacman", "sudo")
	if got, want := hint("linux", "tmux"), "install it with: sudo pacman -S --needed tmux"; got != want {
		t.Fatalf("hint = %q, want %q", got, want)
	}
}

func TestInstallCommandRootDropsSudo(t *testing.T) {
	stubUID(t, 0)
	stubPath(t, "apt-get")
	if got, want := installCommand("linux", "tmux"), "apt-get update && apt-get install -y tmux"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestInstallCommandSkipsAptWithoutSudo(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "apt-get", "brew")
	if got, want := installCommand("linux", "tmux"), "brew install tmux"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestInstallCommandApkAsRootHasNoSudo(t *testing.T) {
	stubUID(t, 0)
	stubPath(t, "apk")
	if got, want := installCommand("linux", "tmux"), "apk add tmux"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestInstallCommandApkAsNonRootTakesSudo(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "apk", "sudo")
	if got, want := installCommand("linux", "tmux"), "sudo apk add tmux"; got != want {
		t.Fatalf("installCommand = %q, want %q", got, want)
	}
}

func TestInstallCommandSkipsApkWithoutSudo(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "apk")
	if got := installCommand("linux", "tmux"); got != "" {
		t.Fatalf("installCommand = %q, want empty", got)
	}
}

func TestHintPrefersOfficialInstallerOverPackageManager(t *testing.T) {
	stubPath(t, "brew")
	got := hint("darwin", "claude")
	if !strings.Contains(got, "claude.ai/install.sh") {
		t.Fatalf("hint = %q, want the official installer", got)
	}
	if strings.Contains(got, "brew install") {
		t.Fatalf("official installer must win over brew, got %q", got)
	}
}

func TestOfficialInstallIsPortable(t *testing.T) {
	if len(official) == 0 {
		t.Fatal("no official installers")
	}
	for name, got := range official {
		if strings.Contains(got, "brew") || strings.Contains(got, "apt") || strings.Contains(got, "winget") {
			t.Errorf("%s install is OS-specific: %s", name, got)
		}
	}
}

func TestCommandIsTheVendorInstaller(t *testing.T) {
	if got := Command("claude"); got != installers(runtime.GOOS)["claude"] {
		t.Fatalf("Command = %q, want the official installer", got)
	}
}

// The curl | bash installers do not run on native Windows, so a hint there
// names the PowerShell installer, a Windows package manager, or nothing.
func TestHintOnWindowsAvoidsUnixInstallers(t *testing.T) {
	stubPath(t, "scoop", "brew", "apt-get")
	if got := hint("windows", "claude"); !strings.Contains(got, "claude.ai/install.ps1") {
		t.Fatalf("hint = %q, want the PowerShell installer", got)
	}
	if got, want := hint("windows", "git"), "install it with: scoop install git"; got != want {
		t.Fatalf("hint = %q, want %q", got, want)
	}
	for _, tool := range []string{"claude", "codex", "gemini", "grok", "hermes", "muse", "opencode", "pi", "agy", "omp", "kiro-cli", "copilot"} {
		if got := hint("windows", tool); strings.Contains(got, "curl") {
			t.Fatalf("hint for %s = %q, want no Unix installer", tool, got)
		}
	}
}

// Every agent CLI the manager ships has a runnable Windows install, so a
// spawn finding a missing one can offer the fix on any platform. psmux and
// the plain-shell rows are not agent CLIs; cmd is documented as unsupported
// on native Windows.
func TestEveryBuiltinAgentHasAWindowsInstall(t *testing.T) {
	agents := []string{"claude", "codex", "gemini", "grok", "hermes", "muse", "opencode", "pi", "agy", "omp", "kiro-cli", "copilot"}
	for _, tool := range agents {
		if got := installers("windows")[tool]; got == "" {
			t.Errorf("%s has no Windows install", tool)
		}
	}
	if got := installers("windows")["cmd"]; got == "" {
		t.Error("cmd lost its npm install, which does run on Windows")
	}
}

// A package-manager line names the tool's own command, so running one for
// a tool we do not know would install whatever package shares that name.
// The hint still says it; only the runnable command is withheld.
func TestCommandWithholdsPackageManagerLines(t *testing.T) {
	stubUID(t, 1)
	stubPath(t, "pacman", "sudo")
	for _, tool := range []string{"tmux", "acme"} {
		if got := Command(tool); got != "" {
			t.Fatalf("Command(%q) = %q, want none", tool, got)
		}
	}
	if got := hint("linux", "tmux"); !strings.Contains(got, "pacman") {
		t.Fatalf("hint = %q, want the package manager still suggested", got)
	}
}

func TestInstallCommandOnDarwinBootstrapsHomebrew(t *testing.T) {
	stubPath(t)
	got := installCommand("darwin", "tmux")
	if got != homebrewSetup+"tmux" {
		t.Fatalf("installCommand = %q, want the Homebrew bootstrap", got)
	}
}

func TestInstallCommandOnWindowsUsesWingetIDs(t *testing.T) {
	stubPath(t, "winget", "scoop")
	if got, want := installCommand("windows", "git"), "winget install --id Git.Git -e"; got != want {
		t.Fatalf("installCommand(git) = %q, want %q", got, want)
	}
	if got, want := installCommand("windows", "pwsh"), "winget install --id Microsoft.PowerShell -e"; got != want {
		t.Fatalf("installCommand(pwsh) = %q, want %q", got, want)
	}
	if got, want := installCommand("windows", "acme"), "scoop install acme"; got != want {
		t.Fatalf("installCommand(acme) = %q, want %q", got, want)
	}
}

func TestWindowsRecipes(t *testing.T) {
	exact := map[string]string{
		"pi":    windowsNpm("--ignore-scripts @earendil-works/pi-coding-agent"),
		"codex": "& ([scriptblock]::Create((irm https://chatgpt.com/codex/install.ps1)))",
		"agy":   "irm https://antigravity.google/cli/install.ps1 | iex",
		"omp":   "irm https://omp.sh/install.ps1 | iex",
		"psmux": "winget install --id marlocarlo.psmux -e",
	}
	for name, want := range exact {
		if got := windowsOfficial[name]; got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	if !strings.Contains(windowsOfficial["opencode"], "@opencode/cli") || strings.Contains(windowsOfficial["opencode"], "opencode-ai") {
		t.Fatalf("opencode = %q, want @opencode/cli", windowsOfficial["opencode"])
	}
	if !strings.Contains(windowsOfficial["cmd"], "command-code") {
		t.Fatalf("cmd = %q, want the npm package", windowsOfficial["cmd"])
	}
}

func TestUnixRecipes(t *testing.T) {
	if got, want := official["pi"], "curl -fsSL https://pi.dev/install.sh | sh"; got != want {
		t.Fatalf("pi = %q, want %q", got, want)
	}
	if !strings.Contains(official["opencode"], "https://opencode.ai/v2/install") || strings.Contains(official["opencode"], "opencode-ai") {
		t.Fatalf("opencode = %q, want the 2.x installer", official["opencode"])
	}
	for _, name := range []string{"gemini", "cmd"} {
		got := official[name]
		if !strings.Contains(got, `--prefix "$HOME/.local"`) || !strings.Contains(got, "node.exe") {
			t.Fatalf("%s = %q, want a prefix fallback that refuses a Windows npm", name, got)
		}
	}
	cmd := official["cmd"]
	head, _, _ := strings.Cut(cmd, "on_windows_path")
	if !strings.Contains(head, "npm install -g command-code") {
		t.Fatalf("cmd lead = %q, want npm install -g command-code", head)
	}
}

func requireDash(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/bin/dash"); err != nil {
		t.Skip("/bin/dash is not installed")
	}
}

func runCommand(t *testing.T, cmd *exec.Cmd, env []string) (string, int) {
	t.Helper()
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	t.Fatal(err)
	return "", 1
}

func TestHintShowsOneLineForScriptRecipes(t *testing.T) {
	for _, tool := range []string{"gemini", "cmd", "opencode"} {
		got := hint("linux", tool)
		if strings.Contains(got, "\n") || got != "install it with: "+shown[tool] {
			t.Fatalf("%s hint = %q, want the one line %q", tool, got, shown[tool])
		}
		if !strings.Contains(official[tool], "publish_file") {
			t.Fatalf("%s recipe = %q, want the npm helper", tool, official[tool])
		}
	}
	if got := hint("windows", "gemini"); got != "install it with: "+windowsOfficial["gemini"] {
		t.Fatalf("windows gemini hint = %q", got)
	}
}
