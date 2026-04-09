<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import MultiSelect from 'primevue/multiselect'
import ProductDetail from './ProductDetail.vue'
import Button from 'primevue/button'
import { getProducts, getDistinctValues, addToBasket, createBasket, addToSharedList, deleteProduct, type Product, type ListParams } from '../api/client'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()
const props = defineProps<{ activeBasketId?: number; activeSharedListId?: number }>()
const emit = defineEmits<{ basketChanged: []; newBasket: [id: number]; sharedListChanged: [] }>()

const products = ref<Product[]>([])
const totalRecords = ref(0)
const loading = ref(false)
const expandedRows = ref<Record<string, boolean>>({})

const search = ref('')
const category = ref('')
const minPrice = ref<number | undefined>()
const maxPrice = ref<number | undefined>()
const sortField = ref('name')
const sortDir = ref('asc')
const page = ref(1)
const pageSize = ref(50)

const nameFilter = ref('')
const producerFilter = ref('')
const countryFilter = ref<string[]>([])
const packagingFilter = ref<string[]>([])
const volumeFilter = ref<string[]>([])
const minAbv = ref<number | undefined>()
const maxAbv = ref<number | undefined>()

const countryOptions = ref<string[]>([])
const packagingOptions = ref<string[]>([])
const volumeOptions = ref<string[]>([])

async function loadFilterOptions() {
  const [countries, pkgs, volumes] = await Promise.all([
    getDistinctValues('country'),
    getDistinctValues('packaging'),
    getDistinctValues('volume'),
  ])
  countryOptions.value = countries
  packagingOptions.value = pkgs
  volumeOptions.value = volumes
}

const categories = [
  { label: 'All', value: '' },
  { label: 'Öl', value: 'Öl' },
  { label: 'Vin', value: 'Vin' },
  { label: 'Sprit', value: 'Sprit' },
  { label: 'Cider & blanddryck', value: 'Cider och blanddrycker' },
  { label: 'Alkoholfritt', value: 'Alkoholfritt' },
]

// Maps column field names to backend sort keys
const colSortMap: Record<string, string> = {
  productNameBold: 'name',
  producerName: 'producer',
  price: 'price',
  alcoholPercentage: 'alcohol',
  volume: 'volume',
  categoryLevel1: 'category',
  country: 'country',
}

function toggleSort(field: string) {
  const backendField = colSortMap[field] || field
  if (sortField.value === backendField) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = backendField
    sortDir.value = 'asc'
  }
  page.value = 1
  loadData()
}

function sortIcon(field: string): string {
  const backendField = colSortMap[field] || field
  if (sortField.value !== backendField) return 'pi pi-sort-alt'
  return sortDir.value === 'asc' ? 'pi pi-sort-amount-up-alt' : 'pi pi-sort-amount-down'
}

function isSorted(field: string): boolean {
  return sortField.value === (colSortMap[field] || field)
}

async function loadData() {
  loading.value = true
  try {
    const params: ListParams = {
      search: search.value || undefined,
      category: category.value || undefined,
      minPrice: minPrice.value,
      maxPrice: maxPrice.value,
      minAbv: minAbv.value,
      maxAbv: maxAbv.value,
      sortBy: sortField.value,
      sortDir: sortDir.value,
      page: page.value,
      pageSize: pageSize.value,
      name: nameFilter.value || undefined,
      producer: producerFilter.value || undefined,
      country: countryFilter.value.length > 0 ? countryFilter.value : undefined,
      packaging: packagingFilter.value.length > 0 ? packagingFilter.value : undefined,
      volume: volumeFilter.value.length > 0 ? volumeFilter.value : undefined,
    }
    const res = await getProducts(params)
    products.value = res.products || []
    totalRecords.value = res.total
  } catch (e) {
    console.error('Failed to load products:', e)
  } finally {
    loading.value = false
  }
}

function onPage(event: { page: number; rows: number }) {
  page.value = event.page + 1
  pageSize.value = event.rows
  loadData()
}

let textTimeout: ReturnType<typeof setTimeout>
watch([search, nameFilter, producerFilter], () => {
  clearTimeout(textTimeout)
  textTimeout = setTimeout(() => {
    page.value = 1
    loadData()
  }, 300)
})

watch([category, minPrice, maxPrice, minAbv, maxAbv, countryFilter, packagingFilter, volumeFilter], () => {
  page.value = 1
  loadData()
})

const errorMsg = ref('')
let errorTimeout: ReturnType<typeof setTimeout>

function showError(msg: string) {
  errorMsg.value = msg
  clearTimeout(errorTimeout)
  errorTimeout = setTimeout(() => { errorMsg.value = '' }, 4000)
}

async function doAddToBasket(productId: string) {
  try {
    let basketId = props.activeBasketId
    if (!basketId) {
      const b = await createBasket(new Date().toLocaleDateString('sv-SE'))
      basketId = b.id
      emit('newBasket', basketId)
    }
    await addToBasket(basketId, productId)
    emit('basketChanged')
  } catch (e: any) {
    const msg = e?.message || String(e)
    if (msg.includes('locked')) {
      showError('This basket is locked and cannot be edited.')
    } else {
      showError(msg)
    }
  }
}

async function doAddToSharedList(productId: string) {
  try {
    if (!props.activeSharedListId) {
      showError('Select a shared list first (open Lists panel).')
      return
    }
    await addToSharedList(props.activeSharedListId, productId)
    emit('sharedListChanged')
  } catch (e: any) {
    showError(e?.message || String(e))
  }
}

async function doDeleteProduct(productId: string) {
  if (!confirm('Delete this product from the database?')) return
  try {
    await deleteProduct(productId)
    loadData()
  } catch (e: any) {
    showError(e?.message || String(e))
  }
}

function resetFilters() {
  search.value = ''
  category.value = ''
  minPrice.value = undefined
  maxPrice.value = undefined
  minAbv.value = undefined
  maxAbv.value = undefined
  nameFilter.value = ''
  producerFilter.value = ''
  countryFilter.value = []
  packagingFilter.value = []
  volumeFilter.value = []
  sortField.value = 'name'
  sortDir.value = 'asc'
  page.value = 1
  loadData()
}

function reload() {
  loadData()
  loadFilterOptions()
}

defineExpose({ reload })

onMounted(() => {
  loadData()
  loadFilterOptions()
})
</script>

<template>
  <div class="card">
    <div class="filters">
      <div class="filter-group">
        <label>Search:</label>
        <InputText v-model="search" placeholder="Name, producer, taste..." />
      </div>
      <div class="filter-group">
        <label>Name:</label>
        <InputText v-model="nameFilter" placeholder="Filter name..." />
      </div>
      <div class="filter-group">
        <label>Producer:</label>
        <InputText v-model="producerFilter" placeholder="Filter producer..." />
      </div>
      <div class="filter-group">
        <label>Category:</label>
        <Select v-model="category" :options="categories" optionLabel="label" optionValue="value" />
      </div>
      <div class="filter-group">
        <label>Country:</label>
        <MultiSelect v-model="countryFilter" :options="countryOptions.map(v => ({ label: v, value: v }))" optionLabel="label" optionValue="value" placeholder="All" :maxSelectedLabels="2" />
      </div>
      <div class="filter-group">
        <label>Volume:</label>
        <MultiSelect v-model="volumeFilter" :options="volumeOptions.map(v => ({ label: v, value: v }))" optionLabel="label" optionValue="value" placeholder="All" :maxSelectedLabels="2" />
      </div>
      <div class="filter-group">
        <label>Packaging:</label>
        <MultiSelect v-model="packagingFilter" :options="packagingOptions.map(v => ({ label: v, value: v }))" optionLabel="label" optionValue="value" placeholder="All" :maxSelectedLabels="2" />
      </div>
      <div class="filter-group">
        <label>Min price:</label>
        <InputText :modelValue="minPrice != null ? String(minPrice) : ''" @update:modelValue="v => minPrice = v ? Number(v) : undefined" placeholder="0" />
      </div>
      <div class="filter-group">
        <label>Max price:</label>
        <InputText :modelValue="maxPrice != null ? String(maxPrice) : ''" @update:modelValue="v => maxPrice = v ? Number(v) : undefined" placeholder="∞" />
      </div>
      <div class="filter-group">
        <label>Min ABV%:</label>
        <InputText :modelValue="minAbv != null ? String(minAbv) : ''" @update:modelValue="v => minAbv = v ? Number(v) : undefined" placeholder="0" />
      </div>
      <div class="filter-group">
        <label>Max ABV%:</label>
        <InputText :modelValue="maxAbv != null ? String(maxAbv) : ''" @update:modelValue="v => maxAbv = v ? Number(v) : undefined" placeholder="∞" />
      </div>
      <div class="filter-actions">
        <Button label="Reset Filters" severity="danger" size="small" @click="resetFilters" />
      </div>
    </div>

    <div v-if="errorMsg" class="error-banner">
      <i class="pi pi-lock" style="font-size: 0.85rem"></i>
      {{ errorMsg }}
      <i class="pi pi-times" style="cursor: pointer; margin-left: auto; font-size: 0.75rem;" @click="errorMsg = ''"></i>
    </div>

    <div class="results-count">{{ totalRecords }} product{{ totalRecords === 1 ? '' : 's' }}</div>

    <DataTable
      :value="products"
      :loading="loading"
      :lazy="true"
      :paginator="true"
      :rows="pageSize"
      :totalRecords="totalRecords"
      :rowsPerPageOptions="[25, 50, 100]"
      @page="onPage"
      v-model:expandedRows="expandedRows"
      dataKey="productId"
      stripedRows
      size="small"
    >
      <Column expander style="width: 3rem" />
      <Column header="" style="width: 50px">
        <template #body="{ data }">
          <img v-if="data.imageUrl" :src="data.imageUrl.replace('_400.', '_60.')" style="height: 40px; border-radius: 3px;" />
        </template>
      </Column>
      <Column field="productNameBold">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('productNameBold') }" @click="toggleSort('productNameBold')">
            Name <i :class="sortIcon('productNameBold')"></i>
          </span>
        </template>
        <template #body="{ data }">
          {{ data.productNameBold }}
          <span v-if="data.productNameThin" style="color: var(--text-muted)"> {{ data.productNameThin }}</span>
          <span v-if="data.isNews" class="badge-news" style="margin-left: 0.5rem">NY</span>
          <span v-if="data.isOrganic" class="badge-organic" style="margin-left: 0.5rem">EKO</span>
          <i v-if="data.note" class="pi pi-comment" style="margin-left: 0.5rem; color: var(--text-faint); font-size: 0.8rem" title="Has note"></i>
        </template>
      </Column>
      <Column field="producerName">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('producerName') }" @click="toggleSort('producerName')">
            Producer <i :class="sortIcon('producerName')"></i>
          </span>
        </template>
      </Column>
      <Column field="price" style="width: 90px">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('price') }" @click="toggleSort('price')">
            Price <i :class="sortIcon('price')"></i>
          </span>
        </template>
        <template #body="{ data }">{{ data.price }} kr</template>
      </Column>
      <Column field="alcoholPercentage" style="width: 80px">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('alcoholPercentage') }" @click="toggleSort('alcoholPercentage')">
            ABV% <i :class="sortIcon('alcoholPercentage')"></i>
          </span>
        </template>
        <template #body="{ data }">{{ data.alcoholPercentage }}%</template>
      </Column>
      <Column field="volume" style="width: 130px">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('volume') }" @click="toggleSort('volume')">
            Volume <i :class="sortIcon('volume')"></i>
          </span>
        </template>
        <template #body="{ data }">{{ data.volumeText }}</template>
      </Column>
      <Column field="categoryLevel1" style="width: 120px">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('categoryLevel1') }" @click="toggleSort('categoryLevel1')">
            Category <i :class="sortIcon('categoryLevel1')"></i>
          </span>
        </template>
      </Column>
      <Column field="country" style="width: 140px">
        <template #header>
          <span class="sort-header" :class="{ active: isSorted('country') }" @click="toggleSort('country')">
            Country <i :class="sortIcon('country')"></i>
          </span>
        </template>
      </Column>
      <Column field="packagingLevel1" style="width: 120px" header="Pkg" />
      <Column header="" :style="{ width: authStore.isAdmin ? '120px' : '90px' }">
        <template #body="{ data }">
          <div style="display: flex; gap: 0.25rem;">
            <Button icon="pi pi-cart-plus" severity="success" rounded size="small"
              title="Add to basket"
              @click="doAddToBasket(data.productId)" />
            <Button icon="pi pi-list" severity="info" rounded size="small"
              title="Add to shared list"
              @click="doAddToSharedList(data.productId)" />
            <Button v-if="authStore.isAdmin" icon="pi pi-trash" severity="danger" rounded size="small"
              title="Delete product"
              @click="doDeleteProduct(data.productId)" />
          </div>
        </template>
      </Column>

      <template #expansion="{ data }">
        <div style="max-width: 700px;">
          <ProductDetail :product="data" @updated="loadData" />
        </div>
      </template>

      <template #empty>
        <div style="text-align: center; padding: 2rem; color: var(--text-muted)">
          No products found. Try syncing some data first.
        </div>
      </template>
    </DataTable>
  </div>
</template>

<style scoped>
.sort-header {
  cursor: pointer;
  user-select: none;
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  color: var(--text-secondary);
  transition: color 0.15s;
}

.sort-header i {
  font-size: 0.7rem;
  color: var(--text-faint);
  transition: color 0.15s;
}

.sort-header:hover {
  color: var(--text);
}

.sort-header:hover i {
  color: var(--text-muted);
}

.sort-header.active {
  color: var(--accent);
}

.sort-header.active i {
  color: var(--accent);
}

.results-count {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-bottom: 0.4rem;
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--warning-light);
  color: #92400e;
  border: 1px solid #f0c36d;
  border-radius: 6px;
  padding: 0.6rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 500;
}
</style>
