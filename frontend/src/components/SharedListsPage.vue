<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import Textarea from 'primevue/textarea'
import Checkbox from 'primevue/checkbox'
import { useAuthStore } from '../stores/auth'
import {
  listSharedLists, createSharedList, getSharedList, deleteSharedList,
  removeFromSharedList, addToSharedList,
  shareSharedList, unshareSharedList, listAllUsers,
  renameSharedList, setSharedListLocked, updateSharedListItemQty,
  type SharedList, type ShareUser,
} from '../api/client'

const authStore = useAuthStore()

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

// Share state
const shareDialogVisible = ref(false)
const allUsers = ref<ShareUser[]>([])
const shareBusy = ref<Set<number>>(new Set())

function isOwner(l: SharedList) {
  return l.userId === authStore.user?.id
}

function ownedLists() {
  return lists.value.filter(l => isOwner(l))
}

function sharedLists() {
  return lists.value.filter(l => !isOwner(l))
}

async function openShareDialog() {
  if (!activeList.value) return
  try {
    allUsers.value = await listAllUsers()
  } catch {
    allUsers.value = []
  }
  shareDialogVisible.value = true
}

function otherUsers(): ShareUser[] {
  return allUsers.value.filter(u => u.userId !== authStore.user?.id)
}

function isSharedWith(userId: number): boolean {
  return (activeList.value?.sharedWith || []).some(u => u.userId === userId)
}

async function toggleShare(userId: number) {
  if (!activeList.value || shareBusy.value.has(userId)) return
  shareBusy.value.add(userId)
  try {
    if (isSharedWith(userId)) {
      await unshareSharedList(activeList.value.id, userId)
    } else {
      await shareSharedList(activeList.value.id, userId)
    }
    activeList.value = await getSharedList(activeList.value.id)
    await loadLists()
  } finally {
    shareBusy.value.delete(userId)
  }
}

async function leaveSharedList() {
  if (!activeList.value || !authStore.user) return
  await unshareSharedList(activeList.value.id, authStore.user.id)
  activeList.value = null
  emit('update:activeId', undefined)
  await loadLists()
}

// Lock
function canToggleLock(l: SharedList) {
  return isOwner(l) || authStore.user?.role === 'admin'
}

async function toggleLock() {
  if (!activeList.value) return
  await setSharedListLocked(activeList.value.id, !activeList.value.locked)
  activeList.value = await getSharedList(activeList.value.id)
  await loadLists()
}

// Rename
const renameDialogVisible = ref(false)
const renameText = ref('')

function openRename() {
  if (!activeList.value) return
  renameText.value = activeList.value.name
  renameDialogVisible.value = true
}

async function doRename() {
  if (!activeList.value || !renameText.value.trim()) return
  await renameSharedList(activeList.value.id, renameText.value.trim())
  renameDialogVisible.value = false
  activeList.value = await getSharedList(activeList.value.id)
  await loadLists()
}

// Qty update
async function updateQty(productId: string, newQty: number) {
  if (!activeList.value) return
  await updateSharedListItemQty(activeList.value.id, productId, newQty)
  await selectList(activeList.value.id)
  await loadLists()
}

const emit = defineEmits<{ 'update:activeId': [id: number | undefined] }>()

const activeItems = computed(() => activeList.value?.items || [])

async function refreshActive() {
  if (activeList.value) {
    await selectList(activeList.value.id)
  }
  await loadLists()
}

// Export / Import JSON
const exportDialogVisible = ref(false)
const exportText = ref('')
const exportCopied = ref(false)
const jsonImportDialogVisible = ref(false)
const jsonImportText = ref('')
const jsonImportBusy = ref(false)
const jsonImportResult = ref<string | null>(null)

function doExport() {
  if (!activeList.value?.items?.length) return
  const data = {
    name: activeList.value.name,
    items: activeList.value.items.map(i => ({
      productId: i.productId,
      quantity: i.quantity,
    })),
  }
  exportText.value = JSON.stringify(data)
  exportCopied.value = false
  exportDialogVisible.value = true
}

async function copyExportText() {
  await navigator.clipboard.writeText(exportText.value)
  exportCopied.value = true
  setTimeout(() => { exportCopied.value = false }, 2000)
}

function openJsonImport() {
  jsonImportText.value = ''
  jsonImportResult.value = null
  jsonImportDialogVisible.value = true
}

async function doJsonImport() {
  jsonImportResult.value = null
  let parsed: { name?: string; items: { productId: string; quantity: number }[] }
  try {
    parsed = JSON.parse(jsonImportText.value)
  } catch {
    jsonImportResult.value = 'Invalid JSON.'
    return
  }
  if (!parsed.items || !Array.isArray(parsed.items) || parsed.items.length === 0) {
    jsonImportResult.value = 'No items found in data.'
    return
  }

  jsonImportBusy.value = true
  try {
    let listId = activeList.value?.id
    if (!listId) {
      const name = parsed.name || `Import ${new Date().toLocaleDateString('sv-SE')}`
      const l = await createSharedList(name)
      listId = l.id
      emit('update:activeId', listId)
    }
    let added = 0
    let failed = 0
    for (const item of parsed.items) {
      if (!item.productId) continue
      try {
        await addToSharedList(listId, item.productId, item.quantity || 1)
        added++
      } catch {
        failed++
      }
    }
    await loadLists()
    await selectList(listId)
    jsonImportDialogVisible.value = false
  } catch (e: any) {
    jsonImportResult.value = `Error: ${e?.message || String(e)}`
  } finally {
    jsonImportBusy.value = false
  }
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

    <template v-else>
      <div class="list-tabs">
        <div
          v-for="l in ownedLists()" :key="l.id"
          class="list-tab" :class="{ active: activeList?.id === l.id }"
          @click="selectList(l.id)"
        >
          <i class="pi pi-list" style="font-size: 0.8rem;"></i>
          <span class="list-name">{{ l.name }}</span>
          <i v-if="l.locked" class="pi pi-lock lock-icon" title="Locked"></i>
          <span class="list-meta">({{ l.itemCount }} items, {{ l.total.toFixed(0) }} kr)</span>
          <i class="pi pi-trash action-icon red" @click.stop="doDelete(l.id)" title="Delete"></i>
        </div>
      </div>

      <div v-if="sharedLists().length > 0">
        <div class="section-label shared-section-label">
          <i class="pi pi-users" style="font-size: 0.75rem"></i> Shared with me
        </div>
        <div class="list-tabs">
          <div
            v-for="l in sharedLists()" :key="l.id"
            class="list-tab shared" :class="{ active: activeList?.id === l.id }"
            @click="selectList(l.id)"
          >
            <i class="pi pi-users" style="font-size: 0.8rem; color: var(--purple);"></i>
            <span class="list-name">{{ l.name }}</span>
            <span class="shared-badge" :title="'Shared by ' + l.ownerName">{{ l.ownerName }}</span>
            <i v-if="l.locked" class="pi pi-lock lock-icon" title="Locked"></i>
            <span class="list-meta">({{ l.itemCount }} items, {{ l.total.toFixed(0) }} kr)</span>
            <i class="pi pi-sign-out action-icon" @click.stop="leaveSharedList" title="Leave shared list" style="font-size: 0.7rem;"></i>
          </div>
        </div>
      </div>
    </template>

    <!-- Active list controls -->
    <div v-if="activeList" class="list-controls">
      <div class="link-bar">
        <label class="link-label">Public link:</label>
        <code class="link-url">{{ getPublicUrl() }}</code>
        <Button :label="copied ? 'Copied!' : 'Copy'" :icon="copied ? 'pi pi-check' : 'pi pi-copy'" size="small" severity="secondary" @click="copyLink" />
        <Button label="Open" icon="pi pi-external-link" size="small" severity="secondary" @click="openPublicUrl" />
        <Button v-if="isOwner(activeList)" label="Share" icon="pi pi-share-alt" size="small" severity="secondary" @click="openShareDialog" />
        <Button v-if="isOwner(activeList)" label="Rename" icon="pi pi-pencil" size="small" severity="secondary" @click="openRename" />
        <Button v-if="canToggleLock(activeList)"
          :label="activeList.locked ? 'Unlock' : 'Lock'"
          :icon="activeList.locked ? 'pi pi-lock-open' : 'pi pi-lock'"
          size="small"
          :severity="activeList.locked ? 'warn' : 'secondary'"
          @click="toggleLock"
        />
        <Button label="Export" icon="pi pi-upload" size="small" severity="secondary" @click="doExport" :disabled="!activeList?.items?.length" />
        <Button label="Import" icon="pi pi-download" size="small" severity="secondary" @click="openJsonImport" :disabled="activeList.locked" />
        <span v-if="activeList.locked" class="locked-badge">
          <i class="pi pi-lock" style="font-size: 0.65rem"></i> Locked
        </span>
      </div>
      <div v-if="activeList.sharedWith && activeList.sharedWith.length > 0" class="shared-with-list">
        Shared with:
        <span v-for="su in activeList.sharedWith" :key="su.userId" class="shared-chip">
          {{ su.username }}
        </span>
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
            <th style="width: 80px;">Qty</th>
            <th style="text-align: right;">Subtotal</th>
            <th>Added by</th>
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
            <td>
              <div class="qty-control">
                <button class="qty-btn" :disabled="activeList?.locked" @click="updateQty(item.productId, item.quantity - 1)">-</button>
                <span class="qty-val">{{ item.quantity }}</span>
                <button class="qty-btn" :disabled="activeList?.locked" @click="updateQty(item.productId, item.quantity + 1)">+</button>
              </div>
            </td>
            <td style="text-align: right; font-weight: 600;">{{ (item.price * item.quantity).toFixed(1) }} kr</td>
            <td><span v-if="item.addedBy" class="added-by">{{ item.addedBy }}</span></td>
            <td>
              <i v-if="!activeList?.locked" class="pi pi-times action-icon red" @click="removeItem(item.productId)" title="Remove"></i>
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <td colspan="7" style="font-weight: 700;">Total</td>
            <td style="text-align: right; font-weight: 700;">{{ activeItems.reduce((s, i) => s + i.price * i.quantity, 0).toFixed(1) }} kr</td>
            <td colspan="2"></td>
          </tr>
        </tfoot>
      </table>
    </div>

    <div v-else-if="activeList" class="empty-state">
      List is empty. Add products from the product table using the <i class="pi pi-list"></i> button.
    </div>

    <!-- Export dialog -->
    <Dialog v-model:visible="exportDialogVisible" modal header="Export List" :style="{ width: '500px', maxWidth: '95vw' }">
      <div class="import-dialog-body">
        <p class="import-hint">Copy this text and paste it on another system to import.</p>
        <Textarea :modelValue="exportText" readonly autoResize style="width: 100%; font-family: monospace; font-size: 0.8rem;" rows="5" />
        <div style="margin-top: 0.75rem;">
          <Button :label="exportCopied ? 'Copied!' : 'Copy to clipboard'" :icon="exportCopied ? 'pi pi-check' : 'pi pi-copy'" size="small" @click="copyExportText" />
        </div>
      </div>
    </Dialog>

    <!-- Import JSON dialog -->
    <Dialog v-model:visible="jsonImportDialogVisible" modal header="Import List" :style="{ width: '500px', maxWidth: '95vw' }">
      <div class="import-dialog-body">
        <p class="import-hint">
          Paste an exported list JSON below.
          {{ activeList ? `Items will be added to "${activeList.name}".` : 'A new list will be created.' }}
        </p>
        <Textarea v-model="jsonImportText" autoResize style="width: 100%; font-family: monospace; font-size: 0.8rem;" rows="5" placeholder='{"name":"...","items":[...]}' />
        <div style="margin-top: 0.75rem; display: flex; align-items: center; gap: 0.5rem;">
          <Button label="Import" icon="pi pi-download" size="small" :disabled="!jsonImportText.trim() || jsonImportBusy" :loading="jsonImportBusy" @click="doJsonImport" />
          <span v-if="jsonImportResult" class="import-result">{{ jsonImportResult }}</span>
        </div>
      </div>
    </Dialog>

    <!-- Share dialog -->
    <Dialog v-model:visible="shareDialogVisible" modal header="Share List" :style="{ width: '360px', maxWidth: '95vw' }">
      <div v-if="activeList" class="share-dialog-body">
        <p class="share-dialog-hint">
          Choose who can see and edit <strong>{{ activeList.name }}</strong>:
        </p>
        <div v-if="otherUsers().length === 0" class="empty-state">No other users to share with.</div>
        <label
          v-for="u in otherUsers()" :key="u.userId"
          class="share-user-row"
          :class="{ busy: shareBusy.has(u.userId) }"
        >
          <Checkbox
            :modelValue="isSharedWith(u.userId)"
            :binary="true"
            :disabled="shareBusy.has(u.userId)"
            @update:modelValue="toggleShare(u.userId)"
          />
          <span class="share-username">{{ u.username }}</span>
          <span v-if="isSharedWith(u.userId)" class="share-status-on">Shared</span>
        </label>
      </div>
    </Dialog>

    <!-- Rename dialog -->
    <Dialog v-model:visible="renameDialogVisible" modal header="Rename List" :style="{ width: '360px', maxWidth: '95vw' }">
      <div class="import-dialog-body">
        <InputText v-model="renameText" style="width: 100%;" @keyup.enter="doRename" />
        <div style="margin-top: 0.75rem;">
          <Button label="Rename" icon="pi pi-check" size="small" :disabled="!renameText.trim()" @click="doRename" />
        </div>
      </div>
    </Dialog>
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

.import-dialog-body {
  padding: 0.25rem 0;
}

.import-hint {
  margin: 0 0 0.75rem 0;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.import-result {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.section-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin: 0.5rem 0 0.25rem 0;
}

.shared-section-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--purple, #9333ea);
}

.list-tab.shared {
  border-color: var(--purple, #9333ea);
  border-style: dashed;
}

.list-tab.shared:hover {
  background: color-mix(in srgb, var(--purple, #9333ea) 8%, transparent);
}

.list-tab.shared.active {
  background: color-mix(in srgb, var(--purple, #9333ea) 12%, transparent);
  border-style: solid;
}

.shared-badge {
  font-size: 0.7rem;
  background: color-mix(in srgb, var(--purple, #9333ea) 15%, transparent);
  color: var(--purple, #9333ea);
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  font-weight: 600;
}

.shared-with-list {
  font-size: 0.8rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
  margin-top: 0.5rem;
}

.shared-chip {
  background: color-mix(in srgb, var(--purple, #9333ea) 12%, transparent);
  color: var(--purple, #9333ea);
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
  font-size: 0.75rem;
  font-weight: 600;
}

.share-dialog-body {
  padding: 0.25rem 0;
}

.share-dialog-hint {
  margin: 0 0 0.75rem 0;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.share-user-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.25rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s;
}

.share-user-row:hover {
  background: var(--bg-muted);
}

.share-user-row.busy {
  opacity: 0.5;
  pointer-events: none;
}

.share-username {
  flex: 1;
  font-size: 0.9rem;
}

.share-status-on {
  font-size: 0.75rem;
  color: var(--purple, #9333ea);
  font-weight: 600;
}

.lock-icon {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.locked-badge {
  font-size: 0.75rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.qty-control {
  display: flex;
  align-items: center;
  gap: 0;
}

.qty-btn {
  width: 22px;
  height: 22px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  cursor: pointer;
  font-size: 0.8rem;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 3px;
  transition: background 0.15s;
}

.qty-btn:hover:not(:disabled) {
  background: var(--bg-muted);
}

.qty-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.qty-val {
  min-width: 24px;
  text-align: center;
  font-size: 0.85rem;
  font-weight: 600;
}

.added-by {
  font-size: 0.8rem;
  color: var(--text-muted);
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
