// web/src/composables/useStreamBuffer.js
// CubeX v9.9.4 真 60FPS 遥测合帧、拍平连接树与悬停冻结缓冲区
import { shallowRef, ref, computed, onMounted, onUnmounted } from 'vue'
import { freezeTelemetryPayload } from '../utils/perf-freeze.js'

export function useStreamBuffer(batchWindowMs = 300) {
  // 1. 强制 shallowRef 承接冻结后的纯数据对象，彻底杜绝 Vue 3 深层 Proxy GC 掉帧
  const nodeBuffer = shallowRef([])
  const rawConnectionList = shallowRef([])
  const flattenedConnectionList = shallowRef([]) // 拍平后的同构行数组 (含设备头行与连接子行)
  const trafficBuffer = shallowRef({ up: 0, down: 0 })

  // 2. 交互状态：展开设备集合、搜索关键词、悬停冻结与手动暂停开关
  const expandedDevices = ref(new Set(['10.0.0.5', '10.0.0.8', '10.0.0.9'])) // 默认展开主要设备
  const connectionSearchQuery = ref('')
  const isPaused = ref(false)
  const isHovered = ref(false)
  const isStressActive = ref(false)

  // 3. 待合帧暂存队列 (非响应式纯变量，性能零开销)
  let pendingNodes = null
  let pendingRawConnections = null
  let pendingTraffic = null
  let pendingSnapshot = null // 悬停/暂停时留档的最新帧快照

  let lastFlushTime = performance.now()
  let rafId = null
  let stressTimer = null

  // 4. 重计算前移至合帧窗口：拍平同构固定行高数组 (每行固定 44px)
  const precomputeFlattenedConnections = (connections, search, expanded) => {
    if (!connections || connections.length === 0) return []

    // A. 搜索过滤
    let filtered = connections
    const q = (search || '').trim().toLowerCase()
    if (q) {
      filtered = connections.filter(c => {
        const host = (c.host || c.metadata?.destinationIP || '').toLowerCase()
        const rule = (c.rule || '').toLowerCase()
        const ip = (c.ip || c.metadata?.sourceIP || '').toLowerCase()
        const out = (c.outbound || '').toLowerCase()
        return host.includes(q) || rule.includes(q) || ip.includes(q) || out.includes(q)
      })
    }

    // B. 按客户端设备 IP 分组
    const groups = new Map()
    for (let i = 0; i < filtered.length; i++) {
      const c = filtered[i]
      const ip = c.ip || c.metadata?.sourceIP || '未知设备'
      let g = groups.get(ip)
      if (!g) {
        g = { ip, conns: [], totalUp: 0, totalDown: 0 }
        groups.set(ip, g)
      }
      g.conns.push(c)
      g.totalUp += (c.uploadSpeed || c.upload || 0)
      g.totalDown += (c.downloadSpeed || c.download || 0)
    }

    // C. 拍平成同构扁平数组 (type: 'device' 与 type: 'conn' 行高同构为 44px)
    const flatList = []
    groups.forEach((group) => {
      const isExp = expanded.has(group.ip)
      flatList.push({
        id: `dev_${group.ip}`,
        type: 'device',
        ip: group.ip,
        totalConns: group.conns.length,
        upRate: group.totalUp,
        downRate: group.totalDown,
        isExpanded: isExp
      })

      if (isExp) {
        for (let j = 0; j < group.conns.length; j++) {
          const item = group.conns[j]
          flatList.push({
            id: `conn_${item.id || item.ip + '_' + j}`,
            type: 'conn',
            ip: group.ip,
            host: item.host || item.metadata?.destinationIP || '直连外网',
            rule: item.rule || 'Match',
            outbound: item.outbound || 'DIRECT',
            upSpeed: item.uploadSpeed || 0,
            downSpeed: item.downloadSpeed || 0
          })
        }
      }
    })

    return Object.freeze(flatList)
  }

  // 5. 核心 rAF 批量合帧调度循环 (300ms 窗口到期一次性替换 shallowRef)
  const flushFrame = (now) => {
    if (now - lastFlushTime >= batchWindowMs) {
      if (!isPaused.value && !isHovered.value) {
        let hasNewData = false

        if (pendingNodes !== null) {
          nodeBuffer.value = freezeTelemetryPayload(pendingNodes)
          pendingNodes = null
          hasNewData = true
        }

        if (pendingRawConnections !== null) {
          rawConnectionList.value = pendingRawConnections
          flattenedConnectionList.value = precomputeFlattenedConnections(
            pendingRawConnections,
            connectionSearchQuery.value,
            expandedDevices.value
          )
          pendingRawConnections = null
          hasNewData = true
        }

        if (pendingTraffic !== null) {
          trafficBuffer.value = freezeTelemetryPayload(pendingTraffic)
          pendingTraffic = null
          hasNewData = true
        }

        // 移开悬停时如果有留档快照，立即刷入
        if (pendingSnapshot !== null) {
          flattenedConnectionList.value = precomputeFlattenedConnections(
            pendingSnapshot,
            connectionSearchQuery.value,
            expandedDevices.value
          )
          pendingSnapshot = null
        }

        lastFlushTime = now
      } else {
        // 处于悬停或手动暂停中：暂存最新原始数据，不触碰响应式 DOM 视图
        if (pendingRawConnections !== null) {
          pendingSnapshot = pendingRawConnections
          pendingRawConnections = null
        }
      }
    }
    rafId = requestAnimationFrame(flushFrame)
  }

  // 6. 切换设备展开/折叠 (零延时重新拍平扁平数组)
  const toggleDeviceExpand = (ip) => {
    if (expandedDevices.value.has(ip)) {
      expandedDevices.value.delete(ip)
    } else {
      expandedDevices.value.add(ip)
    }
    // 立即触发一次拍平渲染
    flattenedConnectionList.value = precomputeFlattenedConnections(
      rawConnectionList.value,
      connectionSearchQuery.value,
      expandedDevices.value
    )
  }

  // 7. 悬停状态切换辅助
  const onMouseEnterTree = () => {
    isHovered.value = true
  }

  const onMouseLeaveTree = () => {
    isHovered.value = false
    // 鼠标移开瞬间立即将最新暂存帧推入视图
    if (pendingSnapshot !== null) {
      flattenedConnectionList.value = precomputeFlattenedConnections(
        pendingSnapshot,
        connectionSearchQuery.value,
        expandedDevices.value
      )
      pendingSnapshot = null
    }
  }

  const togglePause = () => {
    isPaused.value = !isPaused.value
    if (!isPaused.value && pendingSnapshot !== null) {
      flattenedConnectionList.value = precomputeFlattenedConnections(
        pendingSnapshot,
        connectionSearchQuery.value,
        expandedDevices.value
      )
      pendingSnapshot = null
    }
  }

  // 8. 节点网格响应式按行切片 (Zone 2)
  const nodeCols = ref(3) // 默认 3 列
  const updateNodeCols = (containerWidth) => {
    if (containerWidth >= 1280) nodeCols.value = 4
    else if (containerWidth >= 1024) nodeCols.value = 3
    else if (containerWidth >= 640) nodeCols.value = 2
    else nodeCols.value = 1
  }

  const nodeRows = computed(() => {
    const list = nodeBuffer.value || []
    const cols = nodeCols.value || 3
    const rows = []
    for (let i = 0; i < list.length; i += cols) {
      rows.push({
        id: `row_${Math.floor(i / cols)}`,
        nodes: list.slice(i, i + cols)
      })
    }
    return Object.freeze(rows)
  })

  // 9. 内置压测注入器 (window.__CUBEX_STRESS__(500, 3000) 或 ?stress=1)
  const startStressTest = (nodeCount = 500, connCount = 3000) => {
    isStressActive.value = true
    console.log(`[CUBEX-STRESS] 启动极端高压压测: ${nodeCount} 节点 + ${connCount} 连接合帧推送...`)

    // 生成 500 个模拟节点
    const regions = ['HK', 'JP', 'SG', 'US', 'KR', 'DE', 'GB', 'TW']
    const mockNodes = []
    for (let i = 1; i <= nodeCount; i++) {
      const reg = regions[i % regions.length]
      mockNodes.push({
        id: `node_stress_${i}`,
        name: `${reg} 0${(i % 50) + 1} | 极速专线 [${i}]`,
        region: reg,
        delay: Math.floor(15 + Math.random() * 120),
        type: i % 2 === 0 ? 'vless' : 'hysteria2',
        active: i === 1
      })
    }
    pendingNodes = mockNodes

    // 定时每 300ms 注入 3000 条活跃连接
    stressTimer = setInterval(() => {
      const mockConns = []
      const domains = ['youtube.com', 'github.com', 'google.com', 'netflix.com', 'bilibili.com', 'tmdb.org', 'openai.com']
      const rules = ['国际跨境业务协同', '流媒体加速', '应用分流', 'DIRECT', 'AI智能中枢']
      const deviceIPs = []
      for (let d = 1; d <= 25; d++) deviceIPs.push(`10.0.0.${d + 2}`)

      for (let c = 1; c <= connCount; c++) {
        const ip = deviceIPs[c % deviceIPs.length]
        const domain = domains[c % domains.length]
        mockConns.push({
          id: `stress_c_${c}`,
          ip: ip,
          host: `${c}.${domain}`,
          rule: rules[c % rules.length],
          outbound: mockNodes[c % mockNodes.length].name,
          uploadSpeed: Math.floor(Math.random() * 500000),
          downloadSpeed: Math.floor(Math.random() * 5000000)
        })
      }
      pendingRawConnections = mockConns
      pendingTraffic = {
        up: Math.floor(Math.random() * 1024 * 1024 * 20),
        down: Math.floor(Math.random() * 1024 * 1024 * 120)
      }
    }, 300)
  }

  const stopStressTest = () => {
    isStressActive.value = false
    if (stressTimer) {
      clearInterval(stressTimer)
      stressTimer = null
    }
    console.log('[CUBEX-STRESS] 压测已停止。')
  }

  onMounted(() => {
    if (typeof window !== 'undefined') {
      rafId = requestAnimationFrame(flushFrame)

      // 注册全局压测控制钩子
      window.__CUBEX_STRESS__ = (n = 500, c = 3000) => startStressTest(n, c)
      window.__CUBEX_STOP_STRESS__ = () => stopStressTest()
      window.__CUBEX_DOM_COUNT__ = () => {
        const nodeEls = document.querySelectorAll('.virtual-node-card')
        const connEls = document.querySelectorAll('.virtual-conn-row')
        const total = nodeEls.length + connEls.length
        return { nodes: nodeEls.length, connections: connEls.length, total }
      }

      // URL 暗号触发 (?stress=1)
      if (window.location.search.includes('stress=1')) {
        startStressTest(500, 3000)
      }
    }
  })

  onUnmounted(() => {
    if (rafId) {
      cancelAnimationFrame(rafId)
      rafId = null
    }
    if (stressTimer) {
      clearInterval(stressTimer)
      stressTimer = null
    }
  })

  return {
    nodeBuffer,
    flattenedConnectionList,
    trafficBuffer,
    expandedDevices,
    connectionSearchQuery,
    isPaused,
    isHovered,
    isStressActive,
    nodeCols,
    nodeRows,
    updateNodeCols,
    toggleDeviceExpand,
    onMouseEnterTree,
    onMouseLeaveTree,
    togglePause,
    startStressTest,
    stopStressTest
  }
}
