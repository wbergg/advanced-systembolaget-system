<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import { getKeyStatus, refreshKey } from '../api/client'

const hasKey = ref(false)
const refreshing = ref(false)

async function checkStatus() {
  try {
    const res = await getKeyStatus()
    hasKey.value = res.hasKey
  } catch {
    hasKey.value = false
  }
}

async function doRefresh() {
  refreshing.value = true
  try {
    await refreshKey()
    hasKey.value = true
  } catch (e) {
    console.error('Failed to refresh key:', e)
  } finally {
    refreshing.value = false
  }
}

onMounted(checkStatus)
</script>

<template>
  <div class="key-status">
    <span class="dot" :class="hasKey ? 'ok' : 'missing'"></span>
    <span class="label">API {{ hasKey ? 'OK' : 'Missing' }}</span>
    <Button v-if="!hasKey" label="Get Key" icon="pi pi-key" size="small" severity="warn" :loading="refreshing" @click="doRefresh" />
  </div>
</template>

<style scoped>
.key-status {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  margin-right: 0.5rem;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot.ok { background: var(--success); }
.dot.missing { background: var(--danger); }

.label {
  font-size: 0.8rem;
  color: var(--text-muted);
}
</style>
