//go:build linux

package engine

import (
	"os/exec"
	"syscall"
)

// configureSysProcAttr 配置 Linux 专属进程组隔离与父进程崩溃连带清理
func configureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,            // 创建独立进程组，PGID = PID
		Pdeathsig: syscall.SIGKILL, // CubeX 主进程意外崩溃时，内核立即杀死子引擎，杜绝孤儿占坑
	}
}

// signalProcessGroup 向整个子进程组发送信号（负 PID）
func signalProcessGroup(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(-pid, sig)
	if err == syscall.ESRCH {
		return nil // 进程组已不存在，视为成功（幂等）
	}
	return err
}
