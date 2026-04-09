<script setup lang="ts">
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import ProgressBar from 'primevue/progressbar'
import { syncProducts } from '../api/client'

const emit = defineEmits<{ synced: []; cancel: [] }>()

const filters = ref<Record<string, string>>({
  kategori: '',
  forpackning: '',
  'pris-till': '',
  'pris-fran': '',
  land: '',
  q: '',
  butik: '',
})

const syncing = ref(false)
const result = ref<string | null>(null)
const error = ref<string | null>(null)
const progress = ref(0)
const progressText = ref('')

async function doSync() {
  syncing.value = true
  result.value = null
  error.value = null
  progress.value = 0
  progressText.value = ''

  const active: Record<string, string> = {}
  for (const [k, v] of Object.entries(filters.value)) {
    if (v) active[k] = v
  }

  try {
    const res = await syncProducts(active, (p) => {
      progress.value = Math.round((p.page / p.totalPages) * 100)
      progressText.value = `Page ${p.page}/${p.totalPages} (${p.products} products)`
    })
    result.value = `Synced ${res.synced} products`
    progress.value = 100
    progressText.value = ''
    setTimeout(() => emit('synced'), 1500)
  } catch (e: any) {
    error.value = e.message || 'Sync failed'
  } finally {
    syncing.value = false
  }
}
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h3>Sync from Systembolaget</h3>
    </div>
    <div class="sync-dialog">
      <div class="sync-grid">
        <div class="field">
          <label>Category (kategori)</label>
          <InputText v-model="filters.kategori" placeholder="e.g. Öl, Vin, Sprit" />
        </div>
        <div class="field">
          <label>Packaging (forpackning)</label>
          <InputText v-model="filters.forpackning" placeholder="e.g. Burk, Flaska" />
        </div>
        <div class="field">
          <label>Min price (pris-fran)</label>
          <InputText v-model="filters['pris-fran']" placeholder="0" />
        </div>
        <div class="field">
          <label>Max price (pris-till)</label>
          <InputText v-model="filters['pris-till']" placeholder="e.g. 15" />
        </div>
        <div class="field">
          <label>Country (land)</label>
          <InputText v-model="filters.land" placeholder="e.g. Sverige" />
        </div>
        <div class="field">
          <label>Text search (q)</label>
          <InputText v-model="filters.q" placeholder="Free text..." />
        </div>
        <div class="field">
          <label>Store ID (butik)</label>
          <InputText v-model="filters.butik" placeholder="e.g. 0176" />
        </div>
      </div>

      <div v-if="syncing" style="margin-top: 0.75rem;">
        <ProgressBar :value="progress" style="height: 1.25rem;" />
        <small v-if="progressText" class="progress-text">{{ progressText }}</small>
      </div>

      <div class="sync-actions">
        <Button label="Start Sync" icon="pi pi-sync" :loading="syncing" @click="doSync" />
        <Button label="Cancel" severity="secondary" @click="$emit('cancel')" :disabled="syncing" />
        <span v-if="result" class="sync-success">{{ result }}</span>
        <span v-if="error" class="sync-error">{{ error }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sync-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 0.75rem;
}

.sync-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 0.75rem;
  align-items: center;
}

.sync-success {
  color: var(--success);
  font-weight: 600;
  font-size: 0.875rem;
}

.sync-error {
  color: var(--danger);
  font-size: 0.875rem;
}

.progress-text {
  color: var(--text-muted);
  margin-top: 0.25rem;
  display: block;
  font-size: 0.8rem;
}
</style>
