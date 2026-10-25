# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 MikroTik RouterOS v7 极速全自动极简安装与协同脚本
# 适用平台: RouterOS v7.12+ (x86_64 / CHR / RB5009 / CCR2004 / CCR2116 / ARM)
# 核心特性: 全自动识别CPU架构 · 容器扩展自动下载 · 网桥与网段智能探测 · VETH全自动编排
# ═════════════════════════════════════════════════════════════════════════════

{
# 定义双通道同步日志输出函数 (同时输出到 Terminal 终端与 WinBox Log 日志)
:local printMsg do={
    :put $1;
    :log info $1;
};
:local printWarn do={
    :put ("⚠️ " . $1);
    :log warning $1;
};
:local printErr do={
    :put ("❌ " . $1);
    :log error $1;
};

$printMsg "=========================================================================";
$printMsg "🌌 开始执行 CubeX 宿主共生零感网络系统全自动初始化安装流程...";
$printMsg "=========================================================================";

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
            $printErr ("当前硬件架构 (" . $arch . ") 暂不支持自动部署容器，仅支持 x86_64 / arm64 / arm！");
            :error "Unsupported architecture";
        }
    }
}
$printMsg ("✅ [1/5 硬件架构识别] 识别到当前平台: " . $arch . " ➔ 匹配容器与包架构: " . $tarArch);

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 2: 检测是否安装容器扩展包 (没安装则全自动从 MikroTik 官网拉取)
# ─────────────────────────────────────────────────────────────────────────────
:local hasContainer [:len [/system package find where name="container"]];
:if ($hasContainer = 0) do={
    $printWarn "[2/5 容器检测] 未检测到 RouterOS 官方 container 扩展包，正在为您自动下载...";
    :local rosVer [/system resource get version];
    :local spacePos [:find $rosVer " "];
    :local cleanVer $rosVer;
    :if ($spacePos > 0) do={
        :set cleanVer [:pick $rosVer 0 $spacePos];
    }
    :local pkgUrl ("https://download.mikrotik.com/routeros/" . $cleanVer . "/container-" . $cleanVer . "-" . $pkgArch . ".npk");
    :local pkgFile ("container-" . $cleanVer . "-" . $pkgArch . ".npk");
    
    $printMsg ("📥 [CubeX] 正在从 MikroTik 官网下载匹配包: " . $pkgUrl);
    /tool fetch url=$pkgUrl dst-path=$pkgFile;
    
    $printWarn "=========================================================================";
    $printWarn "🎉 container 容器扩展包下载成功！";
    $printWarn "🔔 必须重启路由器才能生效：请在终端执行 [/system reboot] 重启 RouterOS！";
    $printWarn "💡 重启完成后，再次运行本脚本即可无缝继续全自动安装！";
    $printWarn "=========================================================================";
    :error "Container package installed to disk. RouterOS reboot required to activate.";
}

# 检查设备模式 Device Mode (RouterOS v7.4+ 安全限制)
:local dmContainer true;
:do {
    :set dmContainer [/system device-mode get container];
} on-error={};
:if ($dmContainer = false) do={
    $printWarn "[CubeX] 当前 RouterOS 设备模式未激活 container 容器功能，正在尝试自动开启...";
    :do { /system device-mode update container=yes; } on-error={};
    
    :local bName [/system resource get board-name];
    :local isCHR [:typeof [:find $bName "CHR"]];
    
    $printErr "=========================================================================";
    $printErr "RouterOS 容器模式开启申请已下发，必须在 5 分钟内执行硬件冷重启确认：";
    $printErr "-------------------------------------------------------------------------";
    :if ($isCHR != "nil") do={
        $printErr "【虚拟化宿主 (CHR / PVE / ESXi / KVM / Hyper-V)】:";
        $printErr "  ⚠️ 严禁直接在终端执行 /system reboot (系统软重启无法触发硬件确认)！";
        $printErr "  👉 请前往 PVE/ESXi 宿主控制台，对该虚拟机执行【关机/停止 (Stop)】，然后再【启动 (Start)】！";
    } else={
        $printErr "【物理路由器 (RouterBOARD / RB5009 / CCR2004 / CCR2116 / hEX 等)】:";
        $printErr "  👉 请在 5 分钟内长按机身【Reset / Mode】按键直到指示灯闪烁，或直接【拔掉电源插头再插上】冷重启！";
    }
    $printErr "-------------------------------------------------------------------------";
    $printErr "💡 宿主冷重启开机后，再次在终端运行 [/import setup-routeros-v7.rsc] 即可全自动完成！";
    $printErr "=========================================================================";
    :error "Container device-mode confirmation required. Please cold power-cycle within 5 minutes.";
}

$printMsg "✅ [2/5 容器检测] RouterOS 容器功能已就绪！";

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 3: 智能探测网桥 (Bridge) 及局域网网段 (Subnet & Gateway)
# ─────────────────────────────────────────────────────────────────────────────
:local lanBridge "";
:local bridges [/interface bridge find];
:if ([:len $bridges] = 0) do={
    $printWarn "[CubeX] 未检测到任何网桥，正在自动创建主网桥 bridge...";
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
$printMsg ("✅ [3/5 网桥探测] 成功锁定局域网主网桥: " . $lanBridge);

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

# 提取网段前缀 (如 10.0.0.1 ➔ 10.0.0. ; 192.168.88.1 ➔ 192.168.88.)
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
$printMsg ("✅ [3/5 网段与IP分配] 主路由IP: " . $bridgeIP . " | 局域网段: " . $subnetPrefix . "0/" . $bridgeNetmask . " | 分配 CubeX 独立IP: " . $cubexIP);

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 4: 创建 VETH 虚拟网卡并自动加入 Bridge 网桥
# ─────────────────────────────────────────────────────────────────────────────
/interface veth
:if ([:len [find name="veth-cubex"]] = 0) do={
    add name=veth-cubex address=($cubexIP . "/" . $bridgeNetmask) gateway=$bridgeIP comment="CubeX-Gateway-VETH"
    $printMsg ("✅ [4/5 VETH虚拟通道] 已创建 veth-cubex ➔ " . $cubexIP . "/" . $bridgeNetmask);
} else={
    set [find name="veth-cubex"] address=($cubexIP . "/" . $bridgeNetmask) gateway=$bridgeIP
    $printMsg ("✅ [4/5 VETH虚拟通道] 已同步校准 veth-cubex 网段与网关参数");
}

/interface bridge port
# 4.1 挂载 veth-cubex 虚拟通道至主网桥
:if ([:len [find interface="veth-cubex"]] = 0) do={
    add bridge=$lanBridge interface=veth-cubex comment="CubeX-VETH-Bridge-Port"
    $printMsg ("✅ [4/5 网桥挂载] 已将 veth-cubex 挂载至主网桥 " . $lanBridge);
}

# 4.2 多维智能探查并安全挂载局域网 LAN 端口 (完全不依赖单一名称或特定接口清单)
:local wanIfaces [:toarray ""];

# ═════════════════════════════════════════════════════════════════════════════
# 维度 1: 七维多协议深度锁定上行 WAN 口 (严密保护 WAN 口防广播风暴与断网)
# ═════════════════════════════════════════════════════════════════════════════
# 1.1 从 /interface list member 寻找具有 WAN 语义的清单成员 (WAN, wan, internet, isp, uplink等)
:do {
    :foreach m in=[/interface list member find where list~"(?i)(wan|internet|isp|external|outside|uplink)"] do={
        :local wName [/interface list member get $m interface];
        :set wanIfaces ($wanIfaces, $wName);
    }
} on-error={};

# 1.2 从 /ip dhcp-client 寻找活动的 DHCP 客户端上行物理接口
:do {
    :foreach dc in=[/ip dhcp-client find where disabled=no] do={
        :local dcIf [/ip dhcp-client get $dc interface];
        :set wanIfaces ($wanIfaces, $dcIf);
    }
} on-error={};

# 1.3 从 /interface pppoe-client 寻找宽带拨号上行接口及其绑定的底层物理口 (穿透抓底层真实物理口)
:do {
    :foreach pc in=[/interface pppoe-client find] do={
        :local pIf [/interface pppoe-client get $pc interface];
        :local pcName [/interface pppoe-client get $pc name];
        :set wanIfaces ($wanIfaces, $pIf, $pcName);
    }
} on-error={};

# 1.4 从系统默认路由探测实际物理出口 (深度解析 gateway 与 immediate-gw 物理接口)
:do {
    :foreach r in=[/ip route find where dst-address="0.0.0.0/0" and active=yes and routing-table=main] do={
        :local rGate [/ip route get $r gateway];
        :if ([:typeof $rGate] = "str" and [:len $rGate] > 0) do={
            :set wanIfaces ($wanIfaces, $rGate);
        }
        :local imm [/ip route get $r immediate-gw];
        :if ([:typeof $imm] = "str" and [:len $imm] > 0) do={
            :local pctPos [:find $imm "%"];
            :if ([:typeof $pctPos] = "num") do={
                :local ifName [:pick $imm ($pctPos + 1) [:len $imm]];
                :set wanIfaces ($wanIfaces, $ifName);
            } else={
                :set wanIfaces ($wanIfaces, $imm);
            }
        }
    }
} on-error={};

# 1.5 从防火墙出向 NAT 伪装规则提取公网出口接口 (out-interface)
:do {
    :foreach nat in=[/ip firewall nat find where chain="srcnat" and action="masquerade" and disabled=no] do={
        :local outIf [/ip firewall nat get $nat out-interface];
        :if ([:typeof $outIf] = "str" and [:len $outIf] > 0) do={
            :set wanIfaces ($wanIfaces, $outIf);
        }
    }
} on-error={};

# 1.6 扫描蜂窝移动宽带接口 (4G/5G LTE 独立调制解调器)
:do {
    :foreach lte in=[/interface lte find where disabled=no] do={
        :local lName [/interface lte get $lte name];
        :set wanIfaces ($wanIfaces, $lName);
    }
} on-error={};

# 1.7 扫描接口名称或备注中包含 WAN/上行/公网/光猫等关键词的接口
:do {
    :foreach i in=[/interface find where name~"(?i)(wan|internet|isp|modem|uplink)" or comment~"(?i)(wan|internet|isp|modem|uplink|光猫|上联|公网)"] do={
        :local iName [/interface get $i name];
        :set wanIfaces ($wanIfaces, $iName);
    }
} on-error={};

# ═════════════════════════════════════════════════════════════════════════════
# 维度 2: 严格先行排除 WAN 口，只将纯净局域网端口搜集进候选池 (严禁搜集任何 WAN 口)
# ═════════════════════════════════════════════════════════════════════════════
:local lanCandidates [:toarray ""];

# 2.1 扫描内网语义接口清单 (LAN, local, inside, internal, trust, home, 局域网, 内网等) - 严格排除 WAN
:do {
    :foreach m in=[/interface list member find where list~"(?i)(lan|local|internal|inside|trust|home|office|client|user|局域网|内网)"] do={
        :local lName [/interface list member get $m interface];
        :local isWan false;
        :foreach w in=$wanIfaces do={
            :if ($w = $lName) do={ :set isWan true; }
        }
        :if ($isWan = false and $lName != $lanBridge and $lName != "veth-cubex") do={
            :set lanCandidates ($lanCandidates, $lName);
        }
    }
} on-error={};

# 2.2 扫描绑定了本地下行 DHCP-Server 服务的业务接口 - 严格排除 WAN
:do {
    :foreach ds in=[/ip dhcp-server find where disabled=no] do={
        :local dsIf [/ip dhcp-server get $ds interface];
        :local isWan false;
        :foreach w in=$wanIfaces do={
            :if ($w = $dsIf) do={ :set isWan true; }
        }
        :if ($isWan = false and $dsIf != $lanBridge and $dsIf != "veth-cubex") do={
            :set lanCandidates ($lanCandidates, $dsIf);
        }
    }
} on-error={};

# 2.3 全量扫描物理以太网口 (不论叫 ether2、sfp2、LAN1、Trunk、Downlink 还是自定义名) - 严格排除 WAN 绝不误入
:do {
    :foreach eth in=[/interface ethernet find where disabled=no] do={
        :local eName [/interface ethernet get $eth name];
        :local isWan false;
        :foreach w in=$wanIfaces do={
            :if ($w = $eName) do={ :set isWan true; }
        }
        # 若未探查到明确 WAN 且该口为 ether1 / sfp1，按默认习惯保护为上行 WAN 严禁搜集
        :if ([:len $wanIfaces] = 0 and ($eName = "ether1" or $eName = "sfp1" or $eName = "sfp-sfpplus1")) do={
            :set isWan true;
        }
        # 严格只将非 WAN 物理口搜集进候选池
        :if ($isWan = false and $eName != $lanBridge and $eName != "veth-cubex") do={
            :set lanCandidates ($lanCandidates, $eName);
        }
    }
} on-error={};

# 2.4 扫描无线 Wi-Fi / WLAN 局域网接口 (RouterOS v7.13+ wifi 或 legacy wireless) - 严格排除 WAN
:do {
    :foreach w in=[/interface wifi find where disabled=no] do={
        :local wName [/interface wifi get $w name];
        :local isWan false;
        :foreach wan in=$wanIfaces do={
            :if ($wan = $wName) do={ :set isWan true; }
        }
        :if ($isWan = false and $wName != $lanBridge and $wName != "veth-cubex") do={
            :set lanCandidates ($lanCandidates, $wName);
        }
    }
} on-error={};
:do {
    :foreach w in=[/interface wireless find where disabled=no] do={
        :local wName [/interface wireless get $w name];
        :local isWan false;
        :foreach wan in=$wanIfaces do={
            :if ($wan = $wName) do={ :set isWan true; }
        }
        :if ($isWan = false and $wName != $lanBridge and $wName != "veth-cubex") do={
            :set lanCandidates ($lanCandidates, $wName);
        }
    }
} on-error={};

# ═════════════════════════════════════════════════════════════════════════════
# 维度 3: 安全去重并挂载纯净 LAN 候选口至主网桥 (严密过滤 WAN、网桥自身与既有桥端口)
# ═════════════════════════════════════════════════════════════════════════════
:local bridgedCount 0;
:foreach iface in=$lanCandidates do={
    :local skip false;
    
    # 排除主网桥自身与 veth-cubex
    :if ($iface = $lanBridge or $iface = "veth-cubex") do={
        :set skip true;
    }
    
    # 严格排除确认为上行 WAN 的接口
    :if ($skip = false) do={
        :foreach w in=$wanIfaces do={
            :if ($w = $iface) do={ :set skip true; }
        }
    }
    
    # 若设备无任何已知 WAN 标记且该口为首个物理口 ether1/sfp1，按行业标准保护首口为上行
    :if ($skip = false and [:len $wanIfaces] = 0 and ($iface = "ether1" or $iface = "sfp1" or $iface = "sfp-sfpplus1")) do={
        :set skip true;
    }
    
    # 检查是否已加入任何网桥 (避免重复加入或破坏从口)
    :if ($skip = false) do={
        :if ([:len [/interface bridge port find where interface=$iface]] > 0) do={
            :set skip true;
        }
    }
    
    # 确认为纯净局域网端口，安全挂载至主网桥
    :if ($skip = false) do={
        :do {
            /interface bridge port add bridge=$lanBridge interface=$iface comment="CubeX-LAN-Auto-Bridge";
            $printMsg ("✅ [4/5 网桥挂载] 已自动将局域网物理/无线接口 [" . $iface . "] 挂载至主网桥 " . $lanBridge);
            :set bridgedCount ($bridgedCount + 1);
        } on-error={};
    }
}

:if ($bridgedCount = 0) do={
    $printMsg ("ℹ️ [4/5 网桥挂载] 所有非 WAN 局域网端口已就绪挂载于主网桥 " . $lanBridge);
}

# 确保主网桥自身加入 interface list LAN (若存在该列表)
:do {
    :if ([:len [/interface list find where name="LAN"]] > 0) do={
        :if ([:len [/interface list member find where list="LAN" and interface=$lanBridge]] = 0) do={
            /interface list member add list="LAN" interface=$lanBridge comment="CubeX-Bridge-LAN-Member";
        }
    }
} on-error={};

# ─────────────────────────────────────────────────────────────────────────────
# 步骤 5: 配置容器存储、策略路由与自动部署启动 CubeX
# ─────────────────────────────────────────────────────────────────────────────
# 智能探测高速存储磁盘 (优先外部 disk1 / usb1，没有则使用内置根目录)
:local storeDir "cubex";
:local pullDir "cubex/pull";
# 若存在无容器绑定的孤立旧 root-dir 目录残留，自动清理以允许 RouterOS 正常解压
:if ([:len [/container find where root-dir=$storeDir]] = 0) do={
    :do { /file remove [find name=$storeDir]; } on-error={};
}
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
    $printMsg "✅ [5/5 策略路由] 已建立 table-cubex-8228 隔离策略路由表";
}

/ip route
:if ([:len [find routing-table=table-cubex-8228]] = 0) do={
    add dst-address=0.0.0.0/0 gateway=$cubexIP routing-table=table-cubex-8228 check-gateway=ping comment="CubeX-Default-Gateway"
    $printMsg ("✅ [5/5 默认下一跳] 已下发默认策略路由指向 " . $cubexIP . " (支持 ping 毫秒自愈逃生)");
}

# 注入/加固 198.18.0.0/15 Fake-IP 静态回指路由 (覆盖RFC 2544全网段，原生支持“网关留主路由/只改DNS”与“全改旁路由”双模式通吃)
:if ([:len [find dst-address="198.18.0.0/15" and (routing-table=main or routing-table="main")]] = 0) do={
    add dst-address=198.18.0.0/15 gateway=$cubexIP routing-table=main comment="CubeX-FakeIP-Backtrack"
    $printMsg ("✅ [5/5 静态回指路由] 已下发 198.18.0.0/15 静态回指路由指向 " . $cubexIP . " (支持网关留主路由/只改DNS双模式)");
} else={
    :foreach r in=[find where dst-address="198.18.0.0/15"] do={
        :if ([get $r gateway] != $cubexIP) do={
            set $r gateway=$cubexIP comment="CubeX-FakeIP-Backtrack"
            $printMsg ("✅ [5/5 静态回指校准] 已自动校准 198.18.0.0/15 静态回指路由网关为 " . $cubexIP);
        }
    }
}

# Mangle 防死锁豁免与回环放行
/ip firewall mangle
:if ([:len [find comment="CubeX-Bypass-Local"]] = 0) do={
    add chain=prerouting src-address=$cubexIP action=accept comment="CubeX-Bypass-Local"
    add chain=prerouting dst-address=($subnetPrefix . "0/" . $bridgeNetmask) action=accept comment="CubeX-Bypass-LAN"
    $printMsg "✅ [5/5 Mangle防火墙] 已配置防死锁豁免与内网直连放行";
}

# 防恶意/私有 DoT 侧漏阻断 (阻断外发 TCP/UDP 853，防止终端绕过 Fake-IP 进行直连明文/私有探测)
/ip firewall filter
:if ([:len [find comment="CubeX Allow Outbound"]] = 0) do={
    add chain=forward src-address=$cubexIP action=accept comment="CubeX Allow Outbound"
}
:if ([:len [find comment="CubeX Block DoT TCP"]] = 0) do={
    add chain=forward protocol=tcp dst-port=853 action=reject reject-with=tcp-reset comment="CubeX Block DoT TCP"
    add chain=forward protocol=udp dst-port=853 action=reject reject-with=icmp-network-unreachable comment="CubeX Block DoT UDP"
    $printMsg "✅ [5/5 Filter防火墙] 已配置 DoT (端口853) 侧漏防御阻断规则";
}

# 清理既有异常状态的容器残留 (若之前解压失败)
/container
:local failedContainers [find where comment~"CubeX" and (status="error" or status="failed")];
:if ([:len $failedContainers] > 0) do={
    $printWarn "检测到之前解压异常的历史容器，正在为您自动清理重置...";
    :foreach fc in=$failedContainers do={
        remove $fc;
    }
    :delay 2s;
}

# 自动部署安装并启动 CubeX 容器
:local imageTag ("zyeyefyass/cubex:latest-" . $tarArch);
:if ($tarArch = "amd64") do={
    :set imageTag "zyeyefyass/cubex:latest";
}

:if ([:len [find where comment~"CubeX"]] = 0) do={
    :local foundTar "";
    :local defaultTar ("cubex-ros-" . $tarArch . ".tar");
    :if ([:len [/file find name=$defaultTar]] > 0) do={
        :set foundTar $defaultTar;
    } else={
        :foreach f in=[/file find] do={
            :local fName [/file get $f name];
            :if (([:typeof [:find $fName $tarArch]] != "nil") and ([:typeof [:find $fName ".tar"]] != "nil")) do={
                :set foundTar $fName;
            }
        }
    }
    :if ($foundTar != "") do={
        $printMsg ("📦 [CubeX] 发现本地离线包: " . $foundTar . "，正在快速部署...");
        add file=$foundTar interface=veth-cubex root-dir=$storeDir start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M comment="CubeX-Gateway"
    } else={
        $printMsg ("🌐 [CubeX] 未检测到离线 .tar 包，正在从 Docker 镜像源直接拉取: " . $imageTag);
        add remote-image=$imageTag interface=veth-cubex root-dir=$storeDir start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M comment="CubeX-Gateway"
    }
    $printMsg "⏳ 容器部署指令已下发，正在等待 RouterOS 解压并自动启动 (约 5~10 秒)...";
    
    # 终端实时状态进度监听轮询与自动拉起
    :local isRunning false;
    :delay 5s;
    :for i from=1 to=15 do={
        :local cList [/container find where comment~"CubeX"];
        :if ([:len $cList] > 0) do={
            :local cObj [:pick $cList 0];
            :if ([:len [/container find where comment~"CubeX" and running]] > 0) do={
                :set isRunning true;
                :set i 15;
            } else={
                :do { /container start $cObj; } on-error={};
                :delay 2s;
                :if ([:len [/container find where comment~"CubeX" and running]] > 0) do={
                    :set isRunning true;
                    :set i 15;
                } else={
                    :put ("   ...正在解压部署中 (" . ($i * 2) . "s)");
                }
            }
        } else={
            :delay 2s;
        }
    }
    :if ($isRunning = true) do={
        $printMsg "🎉 恭喜！CubeX 容器已成功解压并处于 RUNNING 极速运行状态！";
    } else={
        $printMsg "ℹ️ 容器解压流程已下发，稍后可在 WinBox [Containers] 中观察并启动。";
    }
} else={
    $printMsg "ℹ️ [CubeX] 检测到已存在 CubeX 容器，已确保其处于开机自启状态并加固故障自愈与内存上限";
    set [find comment~"CubeX"] start-on-boot=yes restart-policy=on-failure memory-high=256M memory-max=384M
}

# 自动清理旧内核/历史残留的 Fake-IP 僵尸连接跟踪 (彻底杜绝 missing fakeip record 刷屏)
:do {
    :foreach c in=[/ip/firewall/connection/find where dst-address in 198.18.0.0/15] do={
        /ip/firewall/connection/remove $c
    }
} on-error={};

$printMsg "=========================================================================";
$printMsg "🎉 恭喜！CubeX 宿主共生零感网络系统全自动初始化已全部圆满完成！";
$printMsg ("👉 控制台 Web 驾驶舱访问地址: http://" . $cubexIP . ":2026");
$printMsg "=========================================================================";
}

