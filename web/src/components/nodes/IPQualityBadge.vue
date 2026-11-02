<template>
  <div class="inline-flex items-center gap-1.5 flex-wrap">
    <!-- IP Type Capsule -->
    <span
      v-if="quality?.ip_type"
      class="text-[10px] px-2 py-0.5 rounded-md font-mono border flex items-center gap-1"
      :class="quality.ip_type === 'Residential' ? 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30' : 'bg-blue-500/10 text-blue-300 border-blue-500/30'"
      :title="quality.ip_type === 'Residential' ? '原生住宅宽带 IP · 业务纯净度高' : '商业数据中心 BGP IP'"
    >
      <span class="w-1.5 h-1.5 rounded-full" :class="quality.ip_type === 'Residential' ? 'bg-emerald-400' : 'bg-blue-400'"></span>
      {{ quality.ip_type === 'Residential' ? '原生住宅' : '商业机房' }}
    </span>

    <!-- Clean / Risk Score -->
    <span
      v-if="quality?.risk_level"
      class="text-[10px] px-1.5 py-0.5 rounded-md font-mono border"
      :class="getRiskClass(quality.risk_level)"
      :title="`欺诈风险评分: ${quality.risk_score}/100`"
    >
      {{ quality.risk_level === 'Clean' ? '极净' : quality.risk_level }}
    </span>

    <!-- Feature Unblock Badges -->
    <template v-if="quality?.features">
      <span
        v-for="feat in quality.features.slice(0, 2)"
        :key="feat"
        class="text-[9px] px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-300 border border-purple-500/20"
      >
        {{ feat }}
      </span>
    </template>
  </div>
</template>

<script setup>
defineProps({
  quality: {
    type: Object,
    default: () => null
  }
})

const getRiskClass = (level) => {
  if (level === 'Clean') return 'bg-cyan-500/10 text-cyan-300 border-cyan-500/30'
  if (level === 'Low') return 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30'
  if (level === 'Medium') return 'bg-amber-500/10 text-amber-300 border-amber-500/30'
  return 'bg-rose-500/10 text-rose-300 border-rose-500/30'
}
</script>
