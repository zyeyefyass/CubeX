package engine

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
)

// tunProbeFunc is overridable in unit tests to simulate missing or permission-denied /dev/net/tun.
var tunProbeFunc = defaultProbeTunDevice

// CheckTunDevice verifies that /dev/net/tun exists and is openable with RW access on Linux.
// Returns (true, "") on non-Linux hosts or when TUN device is ready, or (false, actionableReason) otherwise.
func CheckTunDevice() (bool, string) {
	return tunProbeFunc()
}

func defaultProbeTunDevice() (bool, string) {
	if runtime.GOOS != "linux" {
		return true, ""
	}
	return probeTunDevicePath("/dev/net/tun")
}

func probeTunDevicePath(devPath string) (bool, string) {
	info, err := os.Stat(devPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf(
				"缺少 TUN 虚拟网卡设备 (%s 不存在)。群晖/Docker 请先在宿主机加载驱动 (insmod /lib/modules/tun.ko && mkdir -p /dev/net && mknod /dev/net/tun c 10 200)，并在容器启动参数添加 --device /dev/net/tun:/dev/net/tun --cap-add=NET_ADMIN",
				devPath,
			)
		}
		return false, fmt.Sprintf("检查 TUN 设备 %s 状态失败: %v", devPath, err)
	}
	if info.IsDir() {
		return false, fmt.Sprintf("TUN 设备路径 %s 异常为目录而非字符设备，请移除后重建字符设备 (mknod /dev/net/tun c 10 200)", devPath)
	}

	f, err := os.OpenFile(devPath, os.O_RDWR, 0)
	if err != nil {
		return false, classifyTunOpenError(devPath, err)
	}
	_ = f.Close()
	return true, ""
}

func classifyTunOpenError(devPath string, err error) string {
	low := strings.ToLower(err.Error())
	if errors.Is(err, syscall.ENODEV) || strings.Contains(low, "no such device") {
		return fmt.Sprintf(
			"宿主机内核未激活 TUN 驱动模块 (%s: %v)。群晖/NAS 请在宿主机 SSH 执行 insmod /lib/modules/tun.ko 后重启容器",
			devPath, err,
		)
	}
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || strings.Contains(low, "permission") || strings.Contains(low, "operation not permitted") {
		return fmt.Sprintf(
			"容器缺少操作 %s 的权限 (%v)。请确保 Docker 启动时已挂载 --device /dev/net/tun:/dev/net/tun 并赋予 --cap-add=NET_ADMIN（群晖请勿仅用精简图形向导启动）",
			devPath, err,
		)
	}
	return fmt.Sprintf("无法打开 TUN 设备 %s: %v（请检查 --device /dev/net/tun 与 NET_ADMIN 权限）", devPath, err)
}
