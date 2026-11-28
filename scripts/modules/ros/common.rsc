# CubeX RouterOS v7 Shared Functions & Architecture Detection
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

:local arch [/system resource get architecture-name];
:local pkgArch "";
:local tarArch "";
:local cubexDefaultTarPrefix "CubeX-v9.993-ros-";

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
