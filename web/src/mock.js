// CubeX Web Live Demo Mock Interceptor
// 专为 GitHub Pages (https://zyeyefyass.github.io/CubeX/) 打造的免安装在线交互沙盒

export function initDemoMock() {
  const isGitHubPages = window.location.hostname.includes('github.io');
  const isDemoQuery = window.location.search.includes('demo=1');
  const isLocalFile = window.location.protocol === 'file:';
  const isDemoMode = isGitHubPages || isDemoQuery || isLocalFile || import.meta.env.VITE_DEMO_MODE === 'true';

  if (!isDemoMode) {
    return;
  }

  console.info('[CubeX Demo] 🚀 已激活 GitHub Pages 在线交互沙盒 Mock 引擎！');

  // 本地仿真响应式状态池
  let currentEngine = 'sing-box';
  let memoryRSS = 14.9;
  let cpuPercent = 0.1;
  let activeConnections = 38;

  // 16 台高仿真实景设备
  let devices = [
    {
      id: 'dev-1',
      ip: '10.0.0.3',
      mac: '00:11:32:8A:2F:10',
      hostname: 'fnos-nas',
      alias: '飞牛 fnOS 影音私有云 (NAS)',
      brand: 'Linux',
      mode: 'streaming', // 🎬 影音专属模式
      upSpeed: 120400,
      downSpeed: 8450000,
      totalTraffic: 48590000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-2',
      ip: '10.0.0.194',
      mac: 'E4:5F:01:3B:7C:99',
      hostname: 'ms-9205-uhd',
      alias: '魅视 9205 杜比视界蓝光机',
      brand: 'Apple',
      mode: 'streaming',
      upSpeed: 4200,
      downSpeed: 32500000,
      totalTraffic: 98400000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-3',
      ip: '10.0.0.180',
      mac: '70:89:CC:4D:11:22',
      hostname: 'tcl-98x9c-pro',
      alias: '客厅 TCL 98寸 4K 巨幕电视',
      brand: 'TCL',
      mode: 'streaming',
      upSpeed: 1200,
      downSpeed: 14200000,
      totalTraffic: 32100000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-4',
      ip: '10.0.0.100',
      mac: 'F0:18:98:23:45:67',
      hostname: 'macbook-pro-m3',
      alias: '主力工作站 (MacBook Pro M3 Max)',
      brand: 'Apple',
      mode: 'proxy', // 规则分流模式
      upSpeed: 45000,
      downSpeed: 120000,
      totalTraffic: 14200000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-5',
      ip: '10.0.0.105',
      mac: 'A4:C3:F0:88:99:AA',
      hostname: 'iphone-16-pro',
      alias: 'iPhone 16 Pro Max (晨曦)',
      brand: 'Apple',
      mode: 'proxy',
      upSpeed: 2300,
      downSpeed: 45000,
      totalTraffic: 6500000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-6',
      ip: '10.0.0.108',
      mac: 'BC:D0:74:12:34:56',
      hostname: 'ipad-pro-m4',
      alias: 'iPad Pro 13寸 随身平板',
      brand: 'Apple',
      mode: 'proxy',
      upSpeed: 100,
      downSpeed: 2300,
      totalTraffic: 3200000000,
      online: true,
      lastSeen: '1 分钟前'
    },
    {
      id: 'dev-7',
      ip: '10.0.0.2',
      mac: 'D4:01:6D:FE:ED:01',
      hostname: 'pve-node-01',
      alias: 'PVE 8.x 虚拟化宿主物理节点',
      brand: 'Linux',
      mode: 'direct', // 物理直连
      upSpeed: 120,
      downSpeed: 340,
      totalTraffic: 1200000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-8',
      ip: '10.0.0.1',
      mac: '48:8F:5A:12:34:56',
      hostname: 'routeros-ccr2004',
      alias: 'MikroTik RouterOS v7 核心主路由',
      brand: 'MikroTik',
      mode: 'direct',
      upSpeed: 450000,
      downSpeed: 9800000,
      totalTraffic: 128000000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-9',
      ip: '10.0.0.150',
      mac: 'AC:87:A3:11:22:33',
      hostname: 'ps5-pro-console',
      alias: 'PlayStation 5 Pro 游戏主机',
      brand: 'Sony',
      mode: 'direct',
      upSpeed: 50,
      downSpeed: 200,
      totalTraffic: 89000000000,
      online: true,
      lastSeen: '5 分钟前'
    },
    {
      id: 'dev-10',
      ip: '10.0.0.155',
      mac: '98:B6:E9:77:88:99',
      hostname: 'nintendo-switch-oled',
      alias: '任天堂 Switch OLED 掌机',
      brand: 'Nintendo',
      mode: 'proxy',
      upSpeed: 10,
      downSpeed: 50,
      totalTraffic: 1200000000,
      online: true,
      lastSeen: '10 分钟前'
    },
    {
      id: 'dev-11',
      ip: '10.0.0.201',
      mac: '7C:49:EB:11:33:55',
      hostname: 'mi-gateway-hub',
      alias: '小米中枢网关 (米家自动化核心)',
      brand: 'Xiaomi',
      mode: 'direct',
      upSpeed: 300,
      downSpeed: 400,
      totalTraffic: 450000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-12',
      ip: '10.0.0.205',
      mac: '50:EC:50:88:99:00',
      hostname: 'homeassistant-green',
      alias: 'Home Assistant 智能中枢',
      brand: 'Linux',
      mode: 'proxy',
      upSpeed: 1500,
      downSpeed: 8900,
      totalTraffic: 890000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-13',
      ip: '10.0.0.210',
      mac: '18:B9:05:44:55:66',
      hostname: 'aqara-camera-g3',
      alias: 'Aqara G3 智能摄像机 (客厅)',
      brand: 'Aqara',
      mode: 'direct',
      upSpeed: 145000,
      downSpeed: 1200,
      totalTraffic: 24000000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-14',
      ip: '10.0.0.211',
      mac: '18:B9:05:44:55:77',
      hostname: 'aqara-lock-v100',
      alias: 'Aqara 智能门锁 (玄关)',
      brand: 'Aqara',
      mode: 'direct',
      upSpeed: 50,
      downSpeed: 80,
      totalTraffic: 12000000,
      online: true,
      lastSeen: '刚刚'
    },
    {
      id: 'dev-15',
      ip: '10.0.0.120',
      mac: '00:1E:06:55:66:77',
      hostname: 'thinkpad-x1c',
      alias: 'ThinkPad X1 Carbon (家庭办公本)',
      brand: 'Lenovo',
      mode: 'proxy',
      upSpeed: 1200,
      downSpeed: 34000,
      totalTraffic: 4500000000,
      online: true,
      lastSeen: '3 分钟前'
    },
    {
      id: 'dev-16',
      ip: '10.0.0.250',
      mac: 'B0:4E:26:99:AA:BB',
      hostname: 'sonos-era-300',
      alias: 'Sonos Era 300 空间音频音箱',
      brand: 'Sonos',
      mode: 'proxy',
      upSpeed: 200,
      downSpeed: 320000,
      totalTraffic: 15600000000,
      online: true,
      lastSeen: '刚刚'
    }
  ];

  // 劫持全局 window.fetch
  const originalFetch = window.fetch;
  window.fetch = async (resource, init = {}) => {
    const url = typeof resource === 'string' ? resource : resource.url;
    const method = (init.method || 'GET').toUpperCase();

    // 1. 系统概览与双核状态
    if (url.includes('/api/v1/system/overview') || url.includes('/api/v1/status')) {
      const isStoreQuery = window.location.search.includes('store=1');
      return jsonResponse({
        ok: true,
        version: 'v9.9.0',
        edition: 'Host-Symbiotic AutoPilot',
        activeEngine: currentEngine,
        engineStatus: 'running',
        store_mode: isStoreQuery,
        system: {
          hostname: 'CubeX-LiveDemo',
          platform: 'PVE LXC CT 107 (Alpine 3.21) · 沙盒演示版',
          uptime: '18 天 6 小时 32 分',
          cpuPercent: cpuPercent,
          memoryRSS: `${memoryRSS} MB`,
          memoryHeap: '3.8 MB',
          goroutines: 34,
          failOpenState: 'Armed (3s 保底就绪)',
          doctorStatus: '14/14 满分通过',
          rosSymbiosis: 'Connected (table cubex 回指正常)'
        }
      });
    }

    // 2. 8 柱微光动态遥测数据
    if (url.includes('/api/v1/telemetry/traffic') || url.includes('/api/v1/traffic')) {
      const up = Math.floor(Math.random() * 450000) + 120000;
      const down = Math.floor(Math.random() * 8500000) + 2400000;
      return jsonResponse({
        ok: true,
        traffic: {
          upSpeed: up,
          downSpeed: down,
          upSpeedFormatted: `${(up / 1024).toFixed(1)} KB/s`,
          downSpeedFormatted: `${(down / 1024 / 1024).toFixed(2)} MB/s`,
          connections: activeConnections,
          memoryRSS: `${memoryRSS} MB`,
          cpu: `${cpuPercent}%`,
          latencyBaidu: '9ms',
          latencyGoogle: '38ms',
          latencyCloudflare: '42ms'
        }
      });
    }

    // 3. 局域网全息设备拓扑
    if (url.includes('/api/v1/devices') && method === 'GET') {
      return jsonResponse({
        ok: true,
        count: devices.length,
        devices: devices
      });
    }

    // 4. 设备三态模式切换 (直连 / 规则分流 / 🎬 影音专属)
    if (url.includes('/api/v1/devices/policy') || url.includes('/api/v1/devices/mode')) {
      try {
        const body = JSON.parse(init.body || '{}');
        const dev = devices.find(d => d.ip === body.ip || d.id === body.id || d.mac === body.mac);
        if (dev && body.mode) {
          dev.mode = body.mode;
        }
      } catch (e) {}
      return jsonResponse({ ok: true, message: '设备模式已无感热切换！' });
    }

    // 5. 设备别名修改
    if (url.includes('/api/v1/devices/alias')) {
      try {
        const body = JSON.parse(init.body || '{}');
        const dev = devices.find(d => d.ip === body.ip || d.mac === body.mac);
        if (dev && body.alias) {
          dev.alias = body.alias;
        }
      } catch (e) {}
      return jsonResponse({ ok: true, message: '设备别名已保存' });
    }

    // 6. 双核心一键热切换 (sing-box ⇄ Mihomo)
    if (url.includes('/api/v1/engine/switch')) {
      currentEngine = currentEngine === 'sing-box' ? 'Mihomo' : 'sing-box';
      memoryRSS = currentEngine === 'sing-box' ? 14.9 : 28.4;
      return jsonResponse({
        ok: true,
        activeEngine: currentEngine,
        switchDuration: '234ms',
        message: `双核混动热重载完成！已平滑切换至 ${currentEngine}`
      });
    }

    // 7. 健康体检雷达与 Doctor 全量审计
    if (url.includes('/api/v1/health/radar') || url.includes('/api/v1/health/full-audit')) {
      return jsonResponse({
        ok: true,
        score: 100,
        checks: [
          { name: 'POSIX 原子写盘', status: 'pass', desc: '0 损坏兜底' },
          { name: 'GOMEMLIMIT 软上限', status: 'pass', desc: 'RSS ≤ 14.9MB' },
          { name: 'Conntrack 清道夫', status: 'pass', desc: '陈旧连接秒级清算' },
          { name: '198.18 黑洞防外泄', status: 'pass', desc: '0 伪证书报错' },
          { name: 'Netwatch 逃生看门狗', status: 'pass', desc: '3s Fail-Open' },
          { name: '105 套 MRS/SRS 规则集', status: 'pass', desc: '217heidai 权威去广告' },
          { name: 'RouterOS 宿主回指', status: 'pass', desc: 'table cubex 联通' }
        ]
      });
    }

    // 8. 去广告与规则统计
    if (url.includes('/api/v1/adblock') || url.includes('/api/v1/rules')) {
      return jsonResponse({
        ok: true,
        enabled: true,
        ruleCount: 105,
        totalBlocked: 48920,
        todayBlocked: 3412
      });
    }

    // 9. 一句话添加分流规则
    if (url.includes('/api/v1/dns/rewrite') || url.includes('/api/v1/rules/custom')) {
      return jsonResponse({ ok: true, message: '一句话分流规则已秒级生效！' });
    }

    // 兜底默认响应
    if (url.startsWith('/api/')) {
      return jsonResponse({ ok: true, message: 'Demo Mock OK' });
    }

    // 静态文件或其他请求正常放行
    return originalFetch(resource, init);
  };

  function jsonResponse(data, status = 200) {
    return new Response(JSON.stringify(data), {
      status: status,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
}
