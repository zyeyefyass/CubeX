# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 MikroTik RouterOS v7 极速全自动极简安装与协同脚本
# 适用平台: RouterOS v7.12+ (x86_64 / CHR / RB5009 / CCR2004 / CCR2116 / ARM)
# 核心特性: 全自动识别CPU架构 · 容器扩展自动下载 · 网桥与网段智能探测 · VETH全自动编排
# ═════════════════════════════════════════════════════════════════════════════

:log info "🌌 开始执行 CubeX 宿主共生零感网络系统全自动初始化安装流程...";

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 1: 自动判断 RouterOS 硬件 CPU 架构 (amd64 / arm64 / arm)
# ─────────────────────────────────────────────────────────────────────────────
:local arch [/system resource get architecture-name];
:local pkgArch "";
:local tarArch "";

:if ($arch = "x86_64" or $arch = "x86") do={
    :set pkgArch "x86";
    :set tarArch "amd64";
} else={
    :if ($arch = "arm64") do={
        :set pkgArch "arm64";
        :set tarArch "arm64";
    } else={
        :if ($arch = "arm") do={
            :set pkgArch "arm";
            :set tarArch "arm";
        } else={
            :log error ("❌ [CubeX] 当前硬件架构 (" . $arch . ") 暂不支持自动部署容器，仅支持 x86_64 / arm64 / arm！");
            :error "Unsupported architecture";
        }
    }
}
:log info ("✅ [1/5 硬件架构识别] 识别到当前平台: " . $arch . " ➔ 匹配容器与包架构: " . $tarArch);

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 2: 检测是否安装容器扩展包 (没安装则全自动从 MikroTik 官网拉取)
# ─────────────────────────────────────────────────────────────────────────────
:local hasContainer [:len [/system package find where name="container"]];
:if ($hasContainer = 0) do={
    :log warning "⚠️ [2/5 容器检测] 未检测到 RouterOS 官方 container 扩展包，正在为您自动下载...";
    :local rosVer [/system resource get version];
    :local spacePos [:find $rosVer " "];
    :local cleanVer $rosVer;
    :if ($spacePos > 0) do={
        :set cleanVer [:pick $rosVer 0 $spacePos];
    }
    :local pkgUrl ("https://download.mikrotik.com/routeros/" . $cleanVer . "/container-" . $cleanVer . "-" . $pkgArch . ".npk");
    :local pkgFile ("container-" . $cleanVer . "-" . $pkgArch . ".npk");
    
    :log info ("📥 [CubeX] 正在从 MikroTik 官网下载匹配包: " . $pkgUrl);
    /tool fetch url=$pkgUrl dst-path=$pkgFile;
    
    :log warning "=========================================================================";
    :log warning "🎉 container 容器扩展包下载成功！";
    :log warning "🔔 必须重启路由器才能生效：请在终端执行 [/system reboot] 重启 RouterOS！";
    :log warning "💡 重启完成后，再次运行本脚本即可无缝继续全自动安装！";
    :log warning "=========================================================================";
    :error "Container package installed to disk. RouterOS reboot required to activate.";
}

# 检查设备模式 Device Mode (RouterOS v7.4+ 安全限制)
:local dmContainer [/system device-mode get container];
:if ($dmContainer = false) do={
    :log warning "⚠️ [CubeX] 当前 RouterOS 设备模式未激活 container 容器功能，正在尝试自动开启...";
    :do { /system device-mode update container=yes; } on-error={};
    
    :local bName [/system resource get board-name];
    :local isCHR [:typeof [:find $bName "CHR"]];
    
    :log error "=========================================================================";
    :log error "⚠️ [CubeX] RouterOS 容器模式开启申请已下发，必须在 5 分钟内执行硬件冷重启确认：";
    :log error "-------------------------------------------------------------------------";
    :if ($isCHR != "nil") do={
        :log error "【虚拟化宿主 (CHR / PVE / ESXi / KVM / Hyper-V)】:";
        :log error "  ⚠️ 严禁直接在终端执行 /system reboot (系统软重启无法触发硬件确认)！";
        :log error "  👉 请前往 PVE/ESXi 宿主控制台，对该虚拟机执行【关机/停止 (Stop)】，然后再【启动 (Start)】！";
    } else={
        :log error "【物理路由器 (RouterBOARD / RB5009 / CCR2004 / CCR2116 / hEX 等)】:";
        :log error "  👉 请在 5 分钟内长按机身【Reset / Mode】按键直到指示灯闪烁，或直接【拔掉电源插头再插上】冷重启！";
    }
    :log error "-------------------------------------------------------------------------";
    :log error "💡 宿主冷重启开机后，再次在终端粘贴本脚本即可全自动完成后续全部安装！";
    :log error "=========================================================================";
    :error "Container device-mode confirmation required. Please cold power-cycle within 5 minutes.";
}

:log info "✅ [2/5 容器检测] RouterOS 容器功能已就绪！";

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 3: 智能探测网桥 (Bridge) 及局域网网段 (Subnet & Gateway)
# ─────────────────────────────────────────────────────────────────────────────
:local lanBridge "";
:local bridges [/interface bridge find];
:if ([:len $bridges] = 0) do={
    :log warning "⚠️ [CubeX] 未检测到任何网桥，正在自动创建主网桥 bridge...";
    /interface bridge add name=bridge comment="CubeX-Default-Bridge";
    :set lanBridge "bridge";
} else={
    # 优先匹配名为 bridge / bridge-lan 的主网桥，默认取首个网桥
    :set lanBridge [/interface bridge get [:pick $bridges 0] name];
    :foreach b in=$bridges do={
        :local bName [/interface bridge get $b name];
        :if ($bName = "bridge" or $bName = "bridge-lan" or $bName = "lan-bridge") do={
            :set lanBridge $bName;
        }
    }
}
:log info ("✅ [3/5 网桥探测] 成功锁定局域网主网桥: " . $lanBridge);

# 探测网桥绑定的 IP 地址与网段
:local bridgeAddr "";
:local bridgeIP "";
:local bridgeNetmask "24";
:local ipList [/ip address find interface=$lanBridge];
:if ([:len $ipList] > 0) do={
    :set bridgeAddr [/ip address get [:pick $ipList 0] address];
} else={
    :local allAddrs [/ip address find];
    :if ([:len $allAddrs] > 0) do={
        :set bridgeAddr [/ip address get [:pick $allAddrs 0] address];
    } else={
        :set bridgeAddr "192.168.88.1/24";
    }
}

:local slashPos [:find $bridgeAddr "/"];
:if ($slashPos > 0) do={
    :set bridgeIP [:pick $bridgeAddr 0 $slashPos];
    :set bridgeNetmask [:pick $bridgeAddr ($slashPos + 1) [:len $bridgeAddr]];
} else={
    :set bridgeIP $bridgeAddr;
}

# 提取网段前缀 (如 10.0.0.1 ➔ 10.0.0.)
:local lastDot 0;
:for i from=0 to=([:len $bridgeIP] - 1) do={
    :if ([:pick $bridgeIP $i ($i + 1)] = ".") do={
        :set lastDot $i;
    }
}
:local subnetPrefix [:pick $bridgeIP 0 ($lastDot + 1)];

# 自动为 CubeX 分配该网段内的固定专属 IP
# 默认选用 .11；若主路由自身恰好为 .11，则自动选用 .228
:local cubexIP ($subnetPrefix . "11");
:if ($bridgeIP = $cubexIP) do={
    :set cubexIP ($subnetPrefix . "228");
}
:log info ("✅ [3/5 网段与IP分配] 主路由IP: " . $bridgeIP . " | 局域网段: " . $subnetPrefix . "0/" . $bridgeNetmask . " | 分配 CubeX 独立IP: " . $cubexIP);

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 4: 创建 VETH 虚拟网卡并自动加入 Bridge 网桥
# ─────────────────────────────────────────────────────────────────────────────
/interface veth
:if ([:len [find name="veth-cubex"]] = 0) do={
    add name=veth-cubex address=($cubexIP . "/" . $bridgeNetmask) gateway=$bridgeIP comment="CubeX-Gateway-VETH"
    :log info ("✅ [4/5 VETH虚拟通道] 已创建 veth-cubex ➔ " . $cubexIP . "/" . $bridgeNetmask);
} else={
    set [find name="veth-cubex"] address=($cubexIP . "/" . $bridgeNetmask) gateway=$bridgeIP
    :log info ("✅ [4/5 VETH虚拟通道] 已同步校准 veth-cubex 网段与网关参数");
}

/interface bridge port
:if ([:len [find interface="veth-cubex"]] = 0) do={
    add bridge=$lanBridge interface=veth-cubex comment="CubeX-VETH-Bridge-Port"
    :log info ("✅ [4/5 网桥挂载] 已将 veth-cubex 挂载至主网桥 " . $lanBridge);
}

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 5: 配置容器存储、策略路由与自动部署启动 CubeX
# ─────────────────────────────────────────────────────────────────────────────
# 智能探测高速存储磁盘 (优先外部 disk1 / usb1，没有则使用内置根目录)
:local storeDir "cubex";
:local pullDir "cubex/pull";
:do {
    :if ([:len [/file find name="disk1"]] > 0) do={
        :set storeDir "disk1/cubex";
        :set pullDir "disk1/pull";
    } else={
        :if ([:len [/file find name="usb1"]] > 0) do={
            :set storeDir "usb1/cubex";
            :set pullDir "usb1/pull";
        }
    }
} on-error={};

/container config
set memory-high=256M registry-url="https://registry-1.docker.io" tmpdir=$pullDir

# 创建隔离策略路由表 table-cubex-8228 (防死锁)
/routing table
:if ([:len [find name="table-cubex-8228"]] = 0) do={
    add fib name=table-cubex-8228 comment="CubeX-Proxy-Table"
    :log info "✅ [5/5 策略路由] 已建立 table-cubex-8228 隔离策略路由表";
}

/ip route
:if ([:len [find routing-table=table-cubex-8228]] = 0) do={
    add dst-address=0.0.0.0/0 gateway=$cubexIP routing-table=table-cubex-8228 check-gateway=ping comment="CubeX-Default-Gateway"
    :log info ("✅ [5/5 默认下一跳] 已下发默认策略路由指向 " . $cubexIP . " (支持 ping 毫秒自愈逃生)");
}

# Mangle 防死锁豁免与回环放行
/ip firewall mangle
:if ([:len [find comment="CubeX-Bypass-Local"]] = 0) do={
    add chain=prerouting src-address=$cubexIP action=accept comment="CubeX-Bypass-Local"
    add chain=prerouting dst-address=($subnetPrefix . "0/" . $bridgeNetmask) action=accept comment="CubeX-Bypass-LAN"
    :log info "✅ [5/5 Mangle防火墙] 已配置防死锁豁免与内网直连放行";
}

# 自动部署安装并启动 CubeX 容器
:local imageTag ("zyeyefyass/cubex:latest-" . $tarArch);
:if ($tarArch = "amd64") do={
    :set imageTag "zyeyefyass/cubex:latest";
}

/container
:if ([:len [find name="cubex"]] = 0 and [:len [find name~"CubeX"]] = 0) do={
    :local localTar ("cubex-ros-" . $tarArch . ".tar");
    :if ([:len [/file find name=$localTar]] > 0) do={
        :log info ("📦 [CubeX] 发现本地离线包: " . $localTar . "，正在快速部署...");
        add file=$localTar interface=veth-cubex root-dir=$storeDir start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M comment="CubeX-Gateway"
    } else={
        :log info ("🌐 [CubeX] 正在从 Docker Hub 云端直接拉取镜像: " . $imageTag);
        add remote-image=$imageTag interface=veth-cubex root-dir=$storeDir start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M comment="CubeX-Gateway"
    }
    :log info "⏳ 容器创建指令已下发，请在 WinBox [Containers] 中观察状态，解压完成后将自动启动！";
} else={
    :log info "ℹ️ [CubeX] 检测到已存在 CubeX 容器，已确保其处于开机自启状态并加固故障自愈与内存上限";
    set [find comment~"CubeX"] start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M
}

:log info "=========================================================================";
:log info "🎉 恭喜！CubeX 宿主共生零感网络系统全自动初始化已全部圆满完成！";
:log info ("👉 控制台 Web 驾驶舱访问地址: http://" . $cubexIP . ":2026");
:log info "=========================================================================";
