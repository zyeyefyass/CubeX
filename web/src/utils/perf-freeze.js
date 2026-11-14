// 深度冻结纯展示型遥测数组与对象，阻止 Vue 3 为数百个节点/连接创建深层 Proxy 监听器
export function freezeTelemetryPayload(data) {
  if (!data || typeof data !== 'object') {
    return data;
  }
  if (Object.isFrozen(data)) {
    return data;
  }
  if (Array.isArray(data)) {
    for (let i = 0; i < data.length; i++) {
      freezeTelemetryPayload(data[i]);
    }
    return Object.freeze(data);
  }
  const keys = Object.keys(data);
  for (let i = 0; i < keys.length; i++) {
    const val = data[keys[i]];
    if (val && typeof val === 'object') {
      freezeTelemetryPayload(val);
    }
  }
  return Object.freeze(data);
}
