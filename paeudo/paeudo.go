package paeudo

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"os"
	"os/exec"
)

func Command(cmdStr string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		cerr := ErrNew(err, "cmd running error")
		return "", &cerr
	}
	return out.String(), nil
}

func CommandEnv(cmdStr string, env map[string]string, dir string, arg ...string) (string, *Error) {
	path, err := exec.LookPath(cmdStr)
	var cmd *exec.Cmd
	if err != nil {
		bashstr := ""
		bashstr += cmdStr
		for _, a := range arg {
			bashstr += " "
			bashstr += a
		}
		cmd = exec.Command("sh", "-c", bashstr)
	} else {
		cmd = exec.Command(path, arg...)
	}
	cmd.Dir = dir
	envstr := ""
	for key, value := range env {
		envstr += fmt.Sprintf("%s=%s,", key, value)
	}
	cmd.Env = append(os.Environ(), envstr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cerr := ErrNew(err, string(out))
		return "", &cerr
	} else {
		return string(out), nil
	}
}

func ShellEnv(sh string, env map[string]string, dir string, arg ...string) *Error {
	shpath, err := exec.LookPath(sh)
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("shell: %s doesn't exist", sh))
		return &cerr
	} else {
		cmd := exec.Command(shpath, arg...)
		for key, value := range env {
			envstr := fmt.Sprintf("%s=%s", key, value)
			cmd.Env = append(os.Environ(), envstr)
		}
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			cerr := ErrNew(err, "cmd running error")
			return &cerr
		}
	}
	return nil
}
