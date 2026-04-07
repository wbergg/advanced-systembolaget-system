<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import {
  listSharedLists, createSharedList, getSharedList, deleteSharedList,
  removeFromSharedList,
  type SharedList,
} from '../api/client'

const lists = ref<SharedList[]>([])
const activeList = ref<SharedList | null>(null)
const newListName = ref('')
const loading = ref(false)
const copied = ref(false)

async function loadLists() {
  lists.value = await listSharedLists()
}

async function doCreate() {
  if (!newListName.value.trim()) return
  const l = await createSharedList(newListName.value.trim())
  newListName.value = ''
  await loadLists()
  await selectList(l.id)
}

async function selectList(id: number) {
  loading.value = true
  try {
    activeList.value = await getSharedList(id)
    emit('update:activeId', id)
  } finally {
    loading.value = false
  }
}

async function doDelete(id: number) {
  await deleteSharedList(id)
  if (activeList.value?.id === id) {
    activeList.value = null
    emit('update:activeId', undefined)
  }
  await loadLists()
}

async function removeItem(productId: string) {
  if (!activeList.value) return
  await removeFromSharedList(activeList.value.id, productId)
  await selectList(activeList.value.id)
  await loadLists()
}

function getPublicUrl() {
  return `${window.location.origin}/delad-lista/${activeList.value?.uuid}`
}

async function copyLink() {
  await navigator.clipboard.writeText(getPublicUrl())
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

function openPublicUrl() {
  window.open(getPublicUrl(), '_blank')
}

const emit = defineEmits<{ 'update:activeId': [id: number | undefined] }>()

const activeItems = computed(() => activeList.value?.items || [])

async function refreshActive() {
  if (activeList.value) {
    await selectList(activeList.value.id)
  }
  await loadLists()
}

defineExpose({ loadLists, refreshActive })

onMounted(loadLists)
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h3><i class="pi pi-list" style="margin-right: 0.5rem;"></i>Shared Lists</h3>
      <div class="list-create">
        <InputText v-model="newListName" placeholder="New list name..." size="small" @keyup.enter="doCreate" />
        <Button label="Create" icon="pi pi-plus" size="small" @click="doCreate" :disabled="!newListName.trim()" />
      </div>
    </div>

    <div v-if="lists.length === 0" class="empty-state">
      No shared lists yet. Create one and add products to share.
    </div>

    <div v-else class="list-tabs">
      <div
        v-for="l in lists" :key="l.id"
        class="list-tab" :class="{ active: activeList?.id === l.id }"
        @click="selectList(l.id)"
      >
        <i class="pi pi-list" style="font-size: 0.8rem;"></i>
        <span class="list-name">{{ l.name }}</span>
        <span class="list-meta">({{ l.itemCount }} items)</span>
        <i class="pi pi-trash action-icon red" @click.stop="doDelete(l.id)" title="Delete"></i>
      </div>
    </div>

    <!-- Active list controls -->
    <div v-if="activeList" class="list-controls">
      <div class="link-bar">
        <label class="link-label">Public link:</label>
        <code class="link-url">{{ getPublicUrl() }}</code>
        <Button :label="copied ? 'Copied!' : 'Copy'" :icon="copied ? 'pi pi-check' : 'pi pi-copy'" size="small" severity="secondary" @click="copyLink" />
        <Button label="Open" icon="pi pi-external-link" size="small" severity="secondary" @click="openPublicUrl" />
      </div>
    </div>

    <!-- Items table -->
    <div v-if="activeList && activeItems.length > 0">
      <table class="clean-table">
        <thead>
          <tr>
            <th></th>
            <th>Product</th>
            <th>Category</th>
            <th>ABV%</th>
            <th>Volume</th>
            <th>Price</th>
            <th>Country</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in activeItems" :key="item.productId">
            <td>
              <img v-if="item.imageUrl" :src="item.imageUrl.replace('_400.', '_60.')" class="list-thumb" />
            </td>
            <td>
              <strong>{{ item.productNameBold }}</strong>
              <span v-if="item.productNameThin" class="text-muted"> {{ item.productNameThin }}</span>
              <br><span class="text-small text-muted">{{ item.producerName }}</span>
            </td>
            <td class="text-muted">{{ item.categoryLevel1 }}</td>
            <td>{{ item.alcoholPercentage }}%</td>
            <td>{{ item.volumeText }}</td>
            <td>{{ item.price }} kr</td>
            <td class="text-muted">{{ item.country }}</td>
            <td>
              <i class="pi pi-times action-icon red" @click="removeItem(item.productId)" title="Remove"></i>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="activeList" class="empty-state">
      List is empty. Add products from the product table using the <i class="pi pi-list"></i> button.
    </div>
  </div>

</template>

<style scoped>
.list-create {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.list-tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 0.75rem;
}

.list-tab {
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.list-tab:hover {
  border-color: var(--bg-hover);
  background: var(--bg-muted);
}

.list-tab.active {
  border-color: var(--accent);
  background: var(--accent-light);
}

.list-name {
  font-weight: 600;
}

.list-meta {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.list-controls {
  margin-bottom: 0.75rem;
}

.link-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  padding: 0.5rem 0.75rem;
  background: var(--bg-muted);
  border-radius: 6px;
  border: 1px solid var(--border-light);
}

.link-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.link-url {
  font-size: 0.8rem;
  color: var(--accent);
  background: var(--bg-card);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid var(--border);
  word-break: break-all;
  flex: 1;
  min-width: 0;
}

.list-thumb {
  height: 32px;
  border-radius: 3px;
}

.action-icon {
  font-size: 0.7rem;
  color: var(--text-faint);
  cursor: pointer;
  transition: color 0.2s;
}

.action-icon.red:hover {
  color: var(--danger);
}

.text-muted { color: var(--text-muted); }
.text-small { font-size: 0.8rem; }
.empty-state { color: var(--text-muted); padding: 0.75rem 0; font-size: 0.875rem; }

@media (max-width: 768px) {
  .list-create {
    flex-wrap: wrap;
  }

  .link-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .link-url {
    font-size: 0.7rem;
  }

  .clean-table {
    display: block;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
}
</style>
