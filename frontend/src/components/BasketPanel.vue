<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import Checkbox from 'primevue/checkbox'
import ProductDetail from './ProductDetail.vue'
import { useAuthStore } from '../stores/auth'
import {
  listBaskets, createBasket, getBasket, deleteBasket,
  removeFromBasket, updateBasketItemQty, renameBasket,
  getProduct, shareBasket, unshareBasket, listAllUsers,
  setBasketLocked,
  type Basket, type Product, type ShareUser
} from '../api/client'

const authStore = useAuthStore()
const baskets = ref<Basket[]>([])
const activeBasket = ref<Basket | null>(null)
const newBasketName = ref('')
const editingBasketId = ref<number | null>(null)
const editingName = ref('')
const emit = defineEmits<{ 'update:activeId': [id: number | undefined] }>()
const loading = ref(false)

// Sharing state
const shareDialogVisible = ref(false)
const allUsers = ref<ShareUser[]>([])
const shareBusy = ref<Set<number>>(new Set())

async function loadBaskets() {
  baskets.value = await listBaskets()
}

async function doCreate() {
  if (!newBasketName.value.trim()) return
  const b = await createBasket(newBasketName.value.trim())
  newBasketName.value = ''
  await loadBaskets()
  await selectBasket(b.id)
}

async function selectBasket(id: number) {
  loading.value = true
  try {
    activeBasket.value = await getBasket(id)
    emit('update:activeId', id)
  } finally {
    loading.value = false
  }
}

async function doDelete(id: number) {
  await deleteBasket(id)
  if (activeBasket.value?.id === id) {
    activeBasket.value = null
    emit('update:activeId', undefined)
  }
  await loadBaskets()
}

function startRename(b: Basket) {
  editingBasketId.value = b.id
  editingName.value = b.name
}

async function doRename() {
  if (!editingBasketId.value || !editingName.value.trim()) return
  await renameBasket(editingBasketId.value, editingName.value.trim())
  editingBasketId.value = null
  editingName.value = ''
  await loadBaskets()
  if (activeBasket.value) {
    activeBasket.value = await getBasket(activeBasket.value.id)
  }
}

function cancelRename() {
  editingBasketId.value = null
  editingName.value = ''
}

async function refreshActive() {
  if (activeBasket.value) {
    await selectBasket(activeBasket.value.id)
  }
  await loadBaskets()
}

async function removeItem(productId: string) {
  if (!activeBasket.value) return
  await removeFromBasket(activeBasket.value.id, productId)
  await selectBasket(activeBasket.value.id)
  await loadBaskets()
}

async function updateQty(productId: string, qty: number) {
  if (!activeBasket.value) return
  await updateBasketItemQty(activeBasket.value.id, productId, qty)
  await selectBasket(activeBasket.value.id)
  await loadBaskets()
}

const detailProduct = ref<Product | null>(null)
const detailVisible = ref(false)
const detailLoading = ref(false)

async function openProductDetail(productId: string) {
  detailLoading.value = true
  detailVisible.value = true
  try {
    detailProduct.value = await getProduct(productId)
  } catch (e) {
    console.error('Failed to load product:', e)
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

function isOwner(b: Basket) {
  return b.ownerId === authStore.user?.id
}

function ownBaskets() {
  return baskets.value.filter(b => isOwner(b))
}

function sharedBaskets() {
  return baskets.value.filter(b => !isOwner(b))
}

function canToggleLock(b: Basket) {
  return isOwner(b) || authStore.user?.role === 'admin'
}

async function toggleLock() {
  if (!activeBasket.value) return
  await setBasketLocked(activeBasket.value.id, !activeBasket.value.locked)
  activeBasket.value = await getBasket(activeBasket.value.id)
  await loadBaskets()
}

async function openShareDialog() {
  if (!activeBasket.value) return
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
  return (activeBasket.value?.sharedWith || []).some(u => u.userId === userId)
}

async function toggleShare(userId: number) {
  if (!activeBasket.value || shareBusy.value.has(userId)) return
  shareBusy.value.add(userId)
  try {
    if (isSharedWith(userId)) {
      await unshareBasket(activeBasket.value.id, userId)
    } else {
      await shareBasket(activeBasket.value.id, userId)
    }
    activeBasket.value = await getBasket(activeBasket.value.id)
    await loadBaskets()
  } finally {
    shareBusy.value.delete(userId)
  }
}

async function leaveSharedBasket() {
  if (!activeBasket.value || !authStore.user) return
  await unshareBasket(activeBasket.value.id, authStore.user.id)
  activeBasket.value = null
  emit('update:activeId', undefined)
  await loadBaskets()
}

defineExpose({ loadBaskets, refreshActive })

onMounted(loadBaskets)
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h3>Baskets</h3>
      <div class="basket-create">
        <InputText v-model="newBasketName" placeholder="New basket name..." size="small" @keyup.enter="doCreate" />
        <Button label="Create" icon="pi pi-plus" size="small" @click="doCreate" :disabled="!newBasketName.trim()" />
      </div>
    </div>

    <div v-if="baskets.length === 0" class="empty-state">
      No baskets yet. Create one to start adding products.
    </div>

    <template v-else>
      <!-- Own baskets -->
      <div class="section-label">
        <i class="pi pi-shopping-cart" style="font-size: 0.75rem"></i> My Baskets
      </div>
      <div class="basket-tabs">
        <div v-if="ownBaskets().length === 0" class="empty-state" style="padding: 0.25rem 0;">No baskets yet.</div>
        <div
          v-for="b in ownBaskets()" :key="b.id"
          class="basket-tab" :class="{ active: activeBasket?.id === b.id }"
          @click="selectBasket(b.id)"
        >
          <i class="pi pi-shopping-cart" style="font-size: 0.8rem"></i>
          <template v-if="editingBasketId === b.id">
            <InputText v-model="editingName" size="small" style="width: 120px;" autofocus
              @keyup.enter="doRename" @keyup.escape="cancelRename" @click.stop />
            <i class="pi pi-check action-icon green" @click.stop="doRename" title="Save"></i>
            <i class="pi pi-times action-icon" @click.stop="cancelRename" title="Cancel"></i>
          </template>
          <template v-else>
            <span class="basket-name">{{ b.name }}</span>
            <i class="pi pi-pencil action-icon" @click.stop="startRename(b)" title="Rename"></i>
          </template>
          <i v-if="b.locked" class="pi pi-lock lock-icon" title="Locked"></i>
          <span class="basket-meta">({{ b.itemCount }} items, {{ b.total.toFixed(0) }} kr)</span>
          <i class="pi pi-trash action-icon red" @click.stop="doDelete(b.id)" title="Delete"></i>
        </div>
      </div>

      <!-- Shared baskets -->
      <div v-if="sharedBaskets().length > 0">
        <div class="section-label shared-section-label">
          <i class="pi pi-users" style="font-size: 0.75rem"></i> Shared with me
        </div>
        <div class="basket-tabs">
          <div
            v-for="b in sharedBaskets()" :key="b.id"
            class="basket-tab shared" :class="{ active: activeBasket?.id === b.id }"
            @click="selectBasket(b.id)"
          >
            <i class="pi pi-users" style="font-size: 0.8rem; color: var(--purple);"></i>
            <span class="basket-name">{{ b.name }}</span>
            <span class="shared-badge" :title="'Shared by ' + b.ownerName">{{ b.ownerName }}</span>
            <i v-if="b.locked" class="pi pi-lock lock-icon" title="Locked"></i>
            <span class="basket-meta">({{ b.itemCount }} items, {{ b.total.toFixed(0) }} kr)</span>
            <i class="pi pi-sign-out action-icon" @click.stop="leaveSharedBasket" title="Leave shared basket" style="font-size: 0.7rem;"></i>
          </div>
        </div>
      </div>
    </template>

    <!-- Controls for active basket -->
    <div v-if="activeBasket && (isOwner(activeBasket) || canToggleLock(activeBasket))" class="share-bar">
      <Button v-if="isOwner(activeBasket)" label="Share" icon="pi pi-share-alt" size="small" severity="secondary" @click="openShareDialog" />
      <Button v-if="canToggleLock(activeBasket)"
        :label="activeBasket.locked ? 'Unlock' : 'Lock'"
        :icon="activeBasket.locked ? 'pi pi-lock-open' : 'pi pi-lock'"
        size="small"
        :severity="activeBasket.locked ? 'warn' : 'secondary'"
        @click="toggleLock"
      />
      <span v-if="activeBasket.locked" class="locked-badge">
        <i class="pi pi-lock" style="font-size: 0.65rem"></i> Locked — no edits allowed
      </span>
      <span v-if="activeBasket.sharedWith && activeBasket.sharedWith.length > 0" class="shared-with-list">
        Shared with:
        <span v-for="su in activeBasket.sharedWith" :key="su.userId" class="shared-chip">
          {{ su.username }}
        </span>
      </span>
    </div>

    <div v-if="activeBasket && activeBasket.items && activeBasket.items.length > 0">
      <table class="clean-table">
        <thead>
          <tr>
            <th></th>
            <th>Product</th>
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
          <tr v-for="item in activeBasket.items" :key="item.productId">
            <td class="clickable" @click="openProductDetail(item.productId)">
              <img v-if="item.imageUrl" :src="item.imageUrl.replace('_400.', '_60.')" class="basket-thumb" />
            </td>
            <td>
              <a href="#" @click.prevent="openProductDetail(item.productId)" class="product-link">
                <strong>{{ item.productNameBold }}</strong>{{ ' ' }}<span v-if="item.productNameThin" class="text-muted">{{ item.productNameThin }}</span>
              </a>
              <br><span class="text-small text-muted">{{ item.producerName }}</span>
            </td>
            <td>{{ item.alcoholPercentage }}%</td>
            <td>{{ item.volumeText }}</td>
            <td>{{ item.price }} kr</td>
            <td>
              <div class="qty-control">
                <button class="qty-btn" :disabled="activeBasket?.locked" @click="updateQty(item.productId, item.quantity - 1)">-</button>
                <span class="qty-val">{{ item.quantity }}</span>
                <button class="qty-btn" :disabled="activeBasket?.locked" @click="updateQty(item.productId, item.quantity + 1)">+</button>
              </div>
            </td>
            <td style="text-align: right; font-weight: 600;">{{ (item.price * item.quantity).toFixed(1) }} kr</td>
            <td><span v-if="item.addedBy" class="added-by">{{ item.addedBy }}</span></td>
            <td>
              <i v-if="!activeBasket?.locked" class="pi pi-times action-icon red" @click="removeItem(item.productId)"></i>
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <td colspan="7" style="font-weight: 700;">Total</td>
            <td style="text-align: right; font-weight: 700;">{{ activeBasket.total.toFixed(1) }} kr</td>
            <td></td>
          </tr>
        </tfoot>
      </table>
    </div>

    <div v-else-if="activeBasket" class="empty-state">
      Basket is empty. Add products from the table below.
    </div>

    <!-- Share dialog -->
    <Dialog v-model:visible="shareDialogVisible" modal header="Share Basket" :style="{ width: '360px', maxWidth: '95vw' }">
      <div v-if="activeBasket" class="share-dialog-body">
        <p class="share-dialog-hint">
          Choose who can see and edit <strong>{{ activeBasket.name }}</strong>:
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

    <Dialog v-model:visible="detailVisible" modal :style="{ width: '700px', maxWidth: '95vw' }" :header="detailProduct?.productNameBold || 'Product'">
      <div v-if="detailLoading" class="empty-state" style="padding: 2rem;">Loading...</div>
      <ProductDetail v-else-if="detailProduct" :product="detailProduct" @updated="async () => { detailProduct = await getProduct(detailProduct!.productId) }" />
    </Dialog>
  </div>
</template>

<style scoped>
.basket-create {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.section-label {
  font-size: 0.73rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.4rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.shared-section-label {
  color: var(--purple);
  margin-top: 0.75rem;
}

.basket-tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 0.75rem;
}

.basket-tab {
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

.basket-tab:hover {
  border-color: var(--bg-hover);
  background: var(--bg-muted);
}

.basket-tab.active {
  border-color: var(--accent);
  background: var(--accent-light);
}

.basket-tab.shared {
  border-left: 3px solid var(--purple);
  background: var(--purple-light);
}

.basket-tab.shared:hover {
  background: var(--purple-light);
}

.basket-tab.shared.active {
  border-color: var(--purple);
  border-left: 3px solid var(--purple);
  background: var(--purple-light);
}

.basket-name {
  font-weight: 600;
}

.basket-meta {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.shared-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
  background: var(--purple-light);
  color: var(--purple);
  border-radius: 4px;
  padding: 0.1rem 0.4rem;
  font-size: 0.7rem;
  font-weight: 500;
}

.share-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.shared-with-list {
  font-size: 0.8rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.shared-chip {
  display: inline-flex;
  align-items: center;
  background: var(--purple-light);
  color: var(--purple);
  border-radius: 4px;
  padding: 0.15rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
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
  gap: 0.6rem;
  padding: 0.5rem 0.6rem;
  border-radius: 6px;
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
  font-size: 0.875rem;
  font-weight: 500;
  flex: 1;
}

.share-status-on {
  font-size: 0.7rem;
  color: var(--purple);
  font-weight: 500;
}

.lock-icon {
  font-size: 0.7rem;
  color: var(--warning);
}

.locked-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: var(--warning-light);
  color: #92400e;
  border-radius: 4px;
  padding: 0.15rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
}

.qty-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.added-by {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.action-icon {
  font-size: 0.7rem;
  color: var(--text-faint);
  cursor: pointer;
  transition: color 0.2s;
}

.action-icon:hover {
  color: var(--text-secondary);
}

.action-icon.red:hover {
  color: var(--danger);
}

.action-icon.green {
  color: var(--success);
}

.basket-thumb {
  height: 32px;
  border-radius: 3px;
}

.clickable {
  cursor: pointer;
}

.product-link {
  color: inherit;
  text-decoration: none;
}

.product-link:hover strong {
  text-decoration: underline;
}

.text-muted {
  color: var(--text-muted);
}

.text-small {
  font-size: 0.8rem;
}

.empty-state {
  color: var(--text-muted);
  padding: 0.75rem 0;
  font-size: 0.875rem;
}

.qty-control {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.qty-btn {
  border: 1px solid var(--border);
  background: var(--bg-muted);
  border-radius: 4px;
  width: 26px;
  height: 26px;
  cursor: pointer;
  font-size: 0.85rem;
  font-family: var(--font);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  color: var(--text);
}

.qty-btn:hover {
  background: var(--bg-hover);
}

.qty-val {
  width: 26px;
  text-align: center;
  font-weight: 500;
}

@media (max-width: 768px) {
  .basket-create {
    flex-wrap: wrap;
  }

  .share-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .clean-table {
    display: block;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .basket-tabs {
    flex-direction: column;
  }
}
</style>
