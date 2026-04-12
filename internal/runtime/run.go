package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"docksmith/internal/storage"
	"docksmith/internal/utils"
)

func Run(imageName string, extraEnv []string, cmdOverride []string) error {
	filename := strings.ReplaceAll(imageName, ":", "_") + ".json"
	img, err := storage.LoadImage(filename)
	if err != nil {
		return err
	}

	root, err := os.MkdirTemp("", "docksmith-run-")
	if err != nil {
		return err
	}

	for _, layer := range img.Layers {
		layerFile := strings.ReplaceAll(layer.Digest, ":", "_") + ".tar"
		layerPath := filepath.Join(storage.LayersDir(), layerFile)
		if err := utils.ExtractTar(layerPath, root); err != nil {
			return err
		}
	}

	// Determine CMD
	var cmdArgs []string
	if len(cmdOverride) > 0 {
		cmdArgs = cmdOverride
	} else if len(img.Config.Cmd) == 1 && strings.HasPrefix(img.Config.Cmd[0], "[") {
		var parsed []string
		if err := json.Unmarshal([]byte(img.Config.Cmd[0]), &parsed); err != nil {
			return err
		}
		cmdArgs = parsed
	} else {
		cmdArgs = img.Config.Cmd
	}

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no CMD defined")
	}

	envMap := buildEnvMap(img.Config.Env, extraEnv)
	cmdArgs = substituteEnv(cmdArgs, envMap)
	fmt.Println("Running:", cmdArgs)

	workDir := img.Config.WorkingDir
	if workDir == "" {
		workDir = "/"
	}

	if err := os.MkdirAll(filepath.Join(root, workDir), 0755); err != nil {
		return err
	}

	env := append(img.Config.Env, extraEnv...)

	return RunIsolated(root, workDir, cmdArgs, env)
}

func RunIsolated(root, workDir string, cmdArgs, env []string) error {
	if err := copyEssentials(root); err != nil {
		return fmt.Errorf("setup essentials failed: %v", err)
	}

	cmd := exec.Command("/proc/self/exe", append([]string{"__chroot__", root, workDir}, cmdArgs...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunChroot(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("invalid args")
	}

	root := args[0]
	workDir := args[1]
	cmdArgs := args[2:]

	// Create dirs
	os.MkdirAll(filepath.Join(root, "proc"), 0755)
	os.MkdirAll(filepath.Join(root, "dev"), 0755)
	os.MkdirAll(filepath.Join(root, "sys"), 0755)
	os.MkdirAll(filepath.Join(root, "tmp"), 01777)

	// Make mounts private BEFORE chroot
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}

	if err := syscall.Chroot(root); err != nil {
		return err
	}

	if err := syscall.Chdir(workDir); err != nil {
		syscall.Chdir("/")
	}

	syscall.Mount("proc", "/proc", "proc", 0, "")

	cmd := exec.Command("/bin/sh", "-c", strings.Join(cmdArgs, " "))
	cmd.Env = []string{
		"PATH=/usr/bin:/bin:/usr/local/bin",
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

//
// 🔥 CLEAN FIXED VERSION
//
func copyEssentials(root string) error {
	// 🔥 Ensure base dirs exist
	dirs := []string{"bin", "usr/bin", "lib", "lib64"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			return err
		}
	}

	shPath, err := exec.LookPath("sh")
	if err != nil {
		return err
	}

	resolvedSh, err := filepath.EvalSymlinks(shPath)
	if err != nil {
		resolvedSh = shPath
	}

	if err := copyHostBinary(resolvedSh, root); err != nil {
		return err
	}
	if err := copyLibDeps(resolvedSh, root); err != nil {
		return err
	}

	destSh := filepath.Join(root, "bin/sh")

	in, err := os.Open(resolvedSh)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(destSh, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(out, in)

	// utilities...

	// Copy utilities
	binaries := []string{
		"/bin/ls",
		"/bin/touch",
		"/bin/mkdir",
		"/bin/rm",
		"/bin/cat",
		"/bin/echo",
	}

	for _, bin := range binaries {
		resolved, err := filepath.EvalSymlinks(bin)
		if err != nil {
			resolved = bin
		}
		_ = copyHostBinary(resolved, root)
		_ = copyLibDeps(resolved, root)
	}

	return nil
}

func copyHostBinary(src, root string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}

	dest := filepath.Join(root, src)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(out, in)
	return os.Chmod(dest, 0755)
}

func copyLibDeps(binary, root string) error {
	out, err := exec.Command("ldd", binary).Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "/") {
				_ = copyHostBinary(f, root)
			}
		}
	}
	return nil
}

func buildEnvMap(imgEnv, extraEnv []string) map[string]string {
	m := make(map[string]string)
	for _, e := range append(imgEnv, extraEnv...) {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func substituteEnv(args []string, envMap map[string]string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		for key, val := range envMap {
			arg = strings.ReplaceAll(arg, "%"+key+"%", val)
		}
		result[i] = arg
	}
	return result
}
