<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import { listUsers, createUser, updateUser, deleteUser, deleteAllProducts, debugSBProbe, createProduct, listArchivedEvents, type AuthUser, type NewProductPayload, type Event } from '../api/client'
import { useAuthStore } from '../stores/auth'
import EventDetail from './EventDetail.vue'

const authStore = useAuthStore()
const emit = defineEmits<{ close: []; productsChanged: [] }>()

const users = ref<AuthUser[]>([])
const error = ref<string | null>(null)

const newUsername = ref('')
const newPassword = ref('')
const newRole = ref('user')
const creating = ref(false)

const editingUser = ref<AuthUser | null>(null)
const editUsername = ref('')
const editPassword = ref('')
const editRole = ref('user')

const roles = [
  { label: 'User', value: 'user' },
  { label: 'Admin', value: 'admin' },
]

async function loadUsers() {
  try {
    users.value = await listUsers()
  } catch (e: any) {
    error.value = e.message
  }
}

async function doCreate() {
  if (!newUsername.value || !newPassword.value) return
  creating.value = true
  error.value = null
  try {
    await createUser(newUsername.value, newPassword.value, newRole.value)
    newUsername.value = ''
    newPassword.value = ''
    newRole.value = 'user'
    await loadUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    creating.value = false
  }
}

function startEdit(u: AuthUser) {
  editingUser.value = u
  editUsername.value = u.username
  editPassword.value = ''
  editRole.value = u.role
}

async function doUpdate() {
  if (!editingUser.value || !editUsername.value) return
  error.value = null
  try {
    await updateUser(editingUser.value.id, {
      username: editUsername.value,
      password: editPassword.value || undefined,
      role: editRole.value,
    })
    editingUser.value = null
    await loadUsers()
  } catch (e: any) {
    error.value = e.message
  }
}

async function doDelete(id: number) {
  error.value = null
  try {
    await deleteUser(id)
    await loadUsers()
  } catch (e: any) {
    error.value = e.message
  }
}

async function doImpersonate(userId: number) {
  try {
    await authStore.impersonate(userId)
  } catch (e: any) {
    error.value = e.message
  }
}

const purging = ref(false)
const purgeResult = ref<string | null>(null)

// SB probe (debug)
const probeNumber = ref('')
const probeCategory = ref('')
const probePackaging = ref('')
const probePriceMin = ref('')
const probePriceMax = ref('')
const probeBusy = ref(false)
const probeResult = ref<any | null>(null)
const probeError = ref<string | null>(null)

async function doProbe() {
  if (!probeNumber.value.trim()) return
  probeBusy.value = true
  probeResult.value = null
  probeError.value = null
  try {
    probeResult.value = await debugSBProbe(probeNumber.value.trim(), {
      'kategori': probeCategory.value,
      'forpackning': probePackaging.value,
      'pris-fran': probePriceMin.value,
      'pris-till': probePriceMax.value,
    })
  } catch (e: any) {
    probeError.value = e?.message || String(e)
  } finally {
    probeBusy.value = false
  }
}

// Manually add a beer
const showAddProduct = ref(false)
const addingProduct = ref(false)
const addProductError = ref<string | null>(null)
const addProductSuccess = ref<string | null>(null)

type ProductForm = Omit<NewProductPayload, 'price' | 'volume' | 'alcoholPercentage' | 'restrictedParcelQuantity'> & {
  price?: string
  volume?: string
  alcoholPercentage?: string
  restrictedParcelQuantity?: string
}

function emptyProductForm(): ProductForm {
  return {
    productId: '',
    productNumber: '',
    productNameBold: '',
    productNameThin: '',
    producerName: '',
    price: '',
    volume: '',
    volumeText: '',
    alcoholPercentage: '',
    country: '',
    categoryLevel1: '',
    categoryLevel2: '',
    categoryLevel3: '',
    assortmentText: '',
    taste: '',
    usage: '',
    isOrganic: false,
    isNews: false,
    packagingLevel1: '',
    assortment: '',
    productLaunchDate: '',
    restrictedParcelQuantity: '',
    vintage: '',
    imageUrl: '',
  }
}

const newProduct = ref<ProductForm>(emptyProductForm())

function toNumber(v: any): number | undefined {
  if (v === '' || v === null || v === undefined) return undefined
  const n = Number(v)
  return Number.isFinite(n) ? n : undefined
}

async function doCreateProduct() {
  addProductError.value = null
  addProductSuccess.value = null
  if (!newProduct.value.productNameBold?.trim()) {
    addProductError.value = 'Name (bold) is required'
    return
  }
  addingProduct.value = true
  try {
    const payload: NewProductPayload = {
      ...newProduct.value,
      productId: newProduct.value.productId?.trim() || undefined,
      productNameThin: newProduct.value.productNameThin || null,
      vintage: newProduct.value.vintage || null,
      price: toNumber(newProduct.value.price),
      volume: toNumber(newProduct.value.volume),
      alcoholPercentage: toNumber(newProduct.value.alcoholPercentage),
      restrictedParcelQuantity: toNumber(newProduct.value.restrictedParcelQuantity) ?? 0,
    }
    const created = await createProduct(payload)
    addProductSuccess.value = `Added "${created.productNameBold}" (id ${created.productId})`
    newProduct.value = emptyProductForm()
    emit('productsChanged')
  } catch (e: any) {
    addProductError.value = e.message || String(e)
  } finally {
    addingProduct.value = false
  }
}

async function doPurgeProducts() {
  if (!confirm('Delete ALL products from the database? This cannot be undone.')) return
  purging.value = true
  purgeResult.value = null
  try {
    const res = await deleteAllProducts()
    purgeResult.value = `Deleted ${res.deleted} products.`
    emit('productsChanged')
  } catch (e: any) {
    error.value = e.message
  } finally {
    purging.value = false
  }
}

// Archived events
const archivedEvents = ref<Event[]>([])
const archivedLoading = ref(false)
const archivedError = ref<string | null>(null)
const activeArchivedId = ref<number | null>(null)

async function loadArchived() {
  archivedLoading.value = true
  archivedError.value = null
  try {
    archivedEvents.value = await listArchivedEvents()
  } catch (e: any) {
    archivedError.value = e?.message || String(e)
  } finally {
    archivedLoading.value = false
  }
}

function openArchived(id: number) {
  activeArchivedId.value = id
}

function backFromArchived() {
  activeArchivedId.value = null
  loadArchived()
}

function formatArchivedAt(s?: string | null): string {
  if (!s) return ''
  try { return new Date(s).toLocaleString() } catch { return s }
}

onMounted(() => { loadUsers(); loadArchived() })
</script>

<template>
  <div v-if="activeArchivedId !== null">
    <EventDetail :eventId="activeArchivedId" @back="backFromArchived" />
  </div>
  <div v-else class="card">
    <div class="card-header">
      <h3>Admin &mdash; Users</h3>
      <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="$emit('close')" />
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>

    <div class="create-form">
      <div class="field">
        <label>Username</label>
        <InputText v-model="newUsername" placeholder="username" size="small" />
      </div>
      <div class="field">
        <label>Password</label>
        <InputText v-model="newPassword" placeholder="password" size="small" />
      </div>
      <div class="field">
        <label>Role</label>
        <Select v-model="newRole" :options="roles" optionLabel="label" optionValue="value" size="small" style="width: 120px;" />
      </div>
      <Button label="Create" icon="pi pi-plus" size="small" :loading="creating" @click="doCreate"
        :disabled="!newUsername || !newPassword" style="align-self: flex-end;" />
    </div>

    <table class="clean-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Username</th>
          <th>Role</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <template v-if="editingUser?.id === u.id">
            <td>{{ u.id }}</td>
            <td>
              <InputText v-model="editUsername" size="small" style="width: 140px;" />
            </td>
            <td>
              <Select v-model="editRole" :options="roles" optionLabel="label" optionValue="value" size="small" style="width: 100px;" />
            </td>
            <td>
              <div class="action-row">
                <InputText v-model="editPassword" placeholder="new password (optional)" size="small" style="width: 160px;" />
                <Button icon="pi pi-check" severity="success" text rounded size="small" @click="doUpdate" title="Save" />
                <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="editingUser = null" title="Cancel" />
              </div>
            </td>
          </template>
          <template v-else>
            <td>{{ u.id }}</td>
            <td style="font-weight: 600;">{{ u.username }}</td>
            <td>
              <span class="role-badge" :class="u.role">{{ u.role }}</span>
            </td>
            <td>
              <div class="action-row">
                <Button icon="pi pi-pencil" severity="secondary" text rounded size="small" @click="startEdit(u)" title="Edit" />
                <Button icon="pi pi-user" severity="info" text rounded size="small"
                  @click="doImpersonate(u.id)" title="Impersonate"
                  :disabled="u.id === authStore.user?.id" />
                <Button icon="pi pi-trash" severity="danger" text rounded size="small"
                  @click="doDelete(u.id)" title="Delete"
                  :disabled="u.id === authStore.user?.id" />
              </div>
            </td>
          </template>
        </tr>
      </tbody>
    </table>

    <div v-if="users.length === 0" class="empty-state">
      No users found.
    </div>

    <div class="probe-zone">
      <h4>Sync probe (debug)</h4>
      <p class="probe-desc">
        Ask the live Systembolaget API about a specific product number and see whether it's returned under your sync filters, and whether any of our client-side flags would reject it.
      </p>
      <div class="probe-form">
        <div class="probe-field">
          <label>Product Nr</label>
          <InputText v-model="probeNumber" placeholder="e.g. 120115" size="small" />
        </div>
        <div class="probe-field">
          <label>Kategori</label>
          <InputText v-model="probeCategory" placeholder="Öl" size="small" />
        </div>
        <div class="probe-field">
          <label>Förpackning</label>
          <InputText v-model="probePackaging" placeholder="Burk" size="small" />
        </div>
        <div class="probe-field">
          <label>Pris från</label>
          <InputText v-model="probePriceMin" placeholder="0" size="small" style="width: 80px;" />
        </div>
        <div class="probe-field">
          <label>Pris till</label>
          <InputText v-model="probePriceMax" placeholder="30" size="small" style="width: 80px;" />
        </div>
        <Button label="Probe" icon="pi pi-search" size="small" :loading="probeBusy" :disabled="!probeNumber.trim()" @click="doProbe" style="align-self: flex-end;" />
      </div>
      <div v-if="probeError" class="error-msg" style="margin-top: 0.5rem;">{{ probeError }}</div>
      <div v-if="probeResult" class="probe-result">
        <div class="probe-summary">
          <div class="probe-line">
            <strong>With filters:</strong>
            <span v-if="probeResult.withFilters?.apiReturnedProduct" class="probe-yes">API returned it</span>
            <span v-else class="probe-no">API did NOT return it</span>
            <span v-if="probeResult.withFilters?.verdict?.wouldRejectClientSide" class="probe-reject">
              Client would reject: {{ probeResult.withFilters.verdict.rejectReasons.join(', ') }}
            </span>
            <span v-else-if="probeResult.withFilters?.apiReturnedProduct" class="probe-accept">Client would keep it</span>
          </div>
          <div class="probe-line">
            <strong>Without filters:</strong>
            <span v-if="probeResult.withoutFilters?.apiReturnedProduct" class="probe-yes">API returned it</span>
            <span v-else class="probe-no">API did NOT return it (hidden/depot-only/unsearchable)</span>
            <span v-if="probeResult.withoutFilters?.verdict?.wouldRejectClientSide" class="probe-reject">
              Client would reject: {{ probeResult.withoutFilters.verdict.rejectReasons.join(', ') }}
            </span>
          </div>
        </div>
        <details class="probe-raw">
          <summary>Raw JSON</summary>
          <pre>{{ JSON.stringify(probeResult, null, 2) }}</pre>
        </details>
      </div>
    </div>

    <div class="add-product-zone">
      <div class="add-product-header">
        <h4>Manually add a beer</h4>
        <Button :label="showAddProduct ? 'Hide form' : 'Show form'"
          :icon="showAddProduct ? 'pi pi-chevron-up' : 'pi pi-chevron-down'"
          severity="secondary" text size="small" @click="showAddProduct = !showAddProduct" />
      </div>
      <p class="add-product-desc">
        Add a product directly to the database with the same fields a sync would set.
        Leave Product ID empty to auto-generate one.
      </p>

      <div v-if="showAddProduct" class="add-product-form">
        <div class="ap-grid">
          <div class="ap-field">
            <label>Product ID</label>
            <InputText v-model="newProduct.productId" placeholder="(auto)" size="small" />
          </div>
          <div class="ap-field">
            <label>Product Number</label>
            <InputText v-model="newProduct.productNumber" placeholder="e.g. 120115" size="small" />
          </div>
          <div class="ap-field">
            <label>Name (bold) *</label>
            <InputText v-model="newProduct.productNameBold" placeholder="Mariestads" size="small" />
          </div>
          <div class="ap-field">
            <label>Name (thin)</label>
            <InputText v-model="newProduct.productNameThin" placeholder="Export" size="small" />
          </div>
          <div class="ap-field">
            <label>Producer</label>
            <InputText v-model="newProduct.producerName" size="small" />
          </div>
          <div class="ap-field">
            <label>Country</label>
            <InputText v-model="newProduct.country" size="small" />
          </div>
          <div class="ap-field">
            <label>Price (SEK)</label>
            <InputText v-model="newProduct.price" type="number" size="small" />
          </div>
          <div class="ap-field">
            <label>Volume (ml)</label>
            <InputText v-model="newProduct.volume" type="number" size="small" />
          </div>
          <div class="ap-field">
            <label>Volume text</label>
            <InputText v-model="newProduct.volumeText" placeholder="33 cl" size="small" />
          </div>
          <div class="ap-field">
            <label>Alcohol %</label>
            <InputText v-model="newProduct.alcoholPercentage" type="number" size="small" />
          </div>
          <div class="ap-field">
            <label>Category 1</label>
            <InputText v-model="newProduct.categoryLevel1" placeholder="Öl" size="small" />
          </div>
          <div class="ap-field">
            <label>Category 2</label>
            <InputText v-model="newProduct.categoryLevel2" size="small" />
          </div>
          <div class="ap-field">
            <label>Category 3</label>
            <InputText v-model="newProduct.categoryLevel3" size="small" />
          </div>
          <div class="ap-field">
            <label>Packaging</label>
            <InputText v-model="newProduct.packagingLevel1" placeholder="Burk" size="small" />
          </div>
          <div class="ap-field">
            <label>Assortment</label>
            <InputText v-model="newProduct.assortment" size="small" />
          </div>
          <div class="ap-field">
            <label>Assortment text</label>
            <InputText v-model="newProduct.assortmentText" size="small" />
          </div>
          <div class="ap-field">
            <label>Launch date</label>
            <InputText v-model="newProduct.productLaunchDate" placeholder="YYYY-MM-DD" size="small" />
          </div>
          <div class="ap-field">
            <label>Vintage</label>
            <InputText v-model="newProduct.vintage" size="small" />
          </div>
          <div class="ap-field">
            <label>Restricted parcel qty</label>
            <InputText v-model="newProduct.restrictedParcelQuantity" type="number" size="small" />
          </div>
          <div class="ap-field ap-field-wide">
            <label>Image URL</label>
            <InputText v-model="newProduct.imageUrl" size="small" />
          </div>
          <div class="ap-field ap-field-wide">
            <label>Taste</label>
            <InputText v-model="newProduct.taste" size="small" />
          </div>
          <div class="ap-field ap-field-wide">
            <label>Usage</label>
            <InputText v-model="newProduct.usage" size="small" />
          </div>
          <div class="ap-field ap-checkboxes">
            <label class="ap-check">
              <Checkbox v-model="newProduct.isOrganic" :binary="true" /> Organic
            </label>
            <label class="ap-check">
              <Checkbox v-model="newProduct.isNews" :binary="true" /> News
            </label>
          </div>
        </div>

        <div v-if="addProductError" class="error-msg" style="margin-top: 0.75rem;">{{ addProductError }}</div>
        <div v-if="addProductSuccess" class="success-msg">{{ addProductSuccess }}</div>

        <div class="ap-actions">
          <Button label="Add product" icon="pi pi-plus" size="small"
            :loading="addingProduct"
            :disabled="!newProduct.productNameBold?.trim()"
            @click="doCreateProduct" />
          <Button label="Reset" icon="pi pi-refresh" severity="secondary" text size="small"
            :disabled="addingProduct"
            @click="newProduct = emptyProductForm(); addProductError = null; addProductSuccess = null" />
        </div>
      </div>
    </div>

    <div class="archived-zone">
      <div class="archived-header">
        <h4>Archived Events</h4>
        <Button icon="pi pi-refresh" severity="secondary" text size="small" :loading="archivedLoading" @click="loadArchived" title="Refresh" />
      </div>
      <p class="archived-desc">
        Events archived by their owner are hidden from all non-admin users but data is preserved here.
      </p>
      <div v-if="archivedError" class="error-msg">{{ archivedError }}</div>
      <table v-if="archivedEvents.length > 0" class="clean-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Owner</th>
            <th>Event date</th>
            <th>Archived</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ev in archivedEvents" :key="ev.id" class="archived-row" @click="openArchived(ev.id)">
            <td style="font-weight: 600;">{{ ev.name }}</td>
            <td>{{ ev.type }}</td>
            <td>{{ ev.ownerName }}</td>
            <td>{{ ev.eventDate || '-' }}</td>
            <td>{{ formatArchivedAt(ev.archivedAt) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!archivedLoading" class="empty-state">No archived events.</div>
    </div>

    <div class="danger-zone">
      <h4>Danger Zone</h4>
      <div class="danger-row">
        <div>
          <strong>Remove all products</strong>
          <p class="danger-desc">Delete every product from the database. Lists referencing them will break.</p>
        </div>
        <Button label="Delete all products" icon="pi pi-trash" severity="danger" size="small" :loading="purging" @click="doPurgeProducts" />
      </div>
      <span v-if="purgeResult" class="purge-result">{{ purgeResult }}</span>
    </div>
  </div>
</template>

<style scoped>
.error-msg {
  color: var(--danger);
  background: var(--danger-light);
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  margin-bottom: 0.75rem;
}

.create-form {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.action-row {
  display: flex;
  gap: 0.25rem;
  align-items: center;
}

.role-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 12px;
}

.role-badge.admin {
  background: var(--purple-light);
  color: var(--purple);
}

.role-badge.user {
  background: var(--accent-light);
  color: var(--accent);
}

.empty-state {
  color: var(--text-muted);
  padding: 0.75rem 0;
  text-align: center;
  font-size: 0.875rem;
}

.archived-zone {
  margin-top: 1.5rem;
  border-top: 1px solid var(--border);
  padding-top: 1rem;
}

.archived-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.archived-header h4 {
  margin: 0;
  font-size: 0.9rem;
}

.archived-desc {
  margin: 0.35rem 0 0.75rem 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.archived-row {
  cursor: pointer;
  transition: background 0.15s;
}

.archived-row:hover {
  background: var(--bg-muted);
}

.danger-zone {
  margin-top: 1.5rem;
  border-top: 2px solid var(--danger);
  padding-top: 1rem;
}

.danger-zone h4 {
  color: var(--danger);
  margin: 0 0 0.75rem 0;
  font-size: 0.9rem;
}

.danger-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.danger-desc {
  margin: 0.15rem 0 0 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.purge-result {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-top: 0.5rem;
  display: block;
}

.probe-zone {
  margin-top: 1.5rem;
  border-top: 1px solid var(--border);
  padding-top: 1rem;
}

.probe-zone h4 {
  margin: 0 0 0.35rem 0;
  font-size: 0.9rem;
}

.probe-desc {
  margin: 0 0 0.75rem 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.probe-form {
  display: flex;
  gap: 0.6rem;
  align-items: flex-end;
  flex-wrap: wrap;
}

.probe-field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.probe-field label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.probe-result {
  margin-top: 0.75rem;
  background: var(--bg-muted);
  border: 1px solid var(--border-light);
  border-radius: 6px;
  padding: 0.6rem 0.75rem;
}

.probe-summary {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.probe-line {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
  font-size: 0.85rem;
}

.probe-yes {
  color: var(--accent);
  font-weight: 600;
}

.probe-no {
  color: var(--danger);
  font-weight: 600;
}

.probe-reject {
  color: var(--danger);
  font-size: 0.8rem;
}

.probe-accept {
  color: var(--accent);
  font-size: 0.8rem;
}

.probe-raw {
  margin-top: 0.6rem;
  font-size: 0.75rem;
}

.probe-raw summary {
  cursor: pointer;
  color: var(--text-secondary);
  font-weight: 600;
}

.add-product-zone {
  margin-top: 1.5rem;
  border-top: 1px solid var(--border);
  padding-top: 1rem;
}

.add-product-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.add-product-header h4 {
  margin: 0;
  font-size: 0.9rem;
}

.add-product-desc {
  margin: 0.35rem 0 0.75rem 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.ap-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.6rem 0.75rem;
}

.ap-field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.ap-field-wide {
  grid-column: span 2;
}

.ap-field label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.ap-checkboxes {
  flex-direction: row;
  align-items: center;
  gap: 1rem;
  align-self: end;
  padding-bottom: 0.35rem;
}

.ap-check {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-weight: 600;
  cursor: pointer;
}

.ap-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.85rem;
}

.success-msg {
  color: var(--accent);
  background: var(--accent-light);
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  margin-top: 0.75rem;
}

.probe-raw pre {
  margin: 0.4rem 0 0 0;
  max-height: 40vh;
  overflow: auto;
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: 4px;
  padding: 0.5rem;
  font-size: 0.72rem;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
