<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getPublicSharedList, type SharedList, type SharedListItem } from '../api/client'

const props = defineProps<{ uuid: string }>()

const list = ref<SharedList | null>(null)
const loading = ref(true)
const error = ref('')
const listView = ref<'grid' | 'list'>('grid')

onMounted(async () => {
  // Hide the main app styling for the public page
  document.body.style.background = '#ffffff'
  const before = document.querySelector('body') as HTMLElement
  if (before) before.classList.add('sb-public')

  try {
    list.value = await getPublicSharedList(props.uuid)
  } catch {
    error.value = 'Listan kunde inte hittas.'
  } finally {
    loading.value = false
  }
})

const items = computed(() => list.value?.items || [])

function slugify(str: string) {
  return str.toLowerCase()
    .replace(/[åä]/g, 'a').replace(/ö/g, 'o').replace(/é/g, 'e')
    .replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

const categorySlugMap: Record<string, string> = {
  'Öl': 'ol',
  'Vin': 'vin',
  'Sprit': 'sprit',
  'Cider och blanddrycker': 'cider-blanddryck',
  'Alkoholfritt': 'alkoholfritt',
}

function productUrl(item: SharedListItem) {
  const catSlug = categorySlugMap[item.categoryLevel1] || slugify(item.categoryLevel1)
  const nameSlug = slugify(item.productNameBold)
  return `https://www.systembolaget.se/produkt/${catSlug}/${nameSlug}-${item.productNumber}/`
}

function imageUrl(item: SharedListItem, size: string) {
  if (!item.imageUrl) return ''
  return item.imageUrl.replace('_400.', `_${size}.`)
}

function tastePreview(item: SharedListItem) {
  if (!item.taste) return ''
  return item.taste.length > 80 ? item.taste.slice(0, 80) + '...' : item.taste
}
</script>

<template>
  <div class="sb-page">
    <!-- Header -->
    <header class="sb-header">
      <div class="sb-header-inner">
        <span class="sb-brand">Advanced Systembolaget System</span>
      </div>
    </header>

    <main class="sb-main">
      <div v-if="loading" class="sb-loading">
        <div class="sb-spinner"></div>
      </div>

      <div v-else-if="error" class="sb-error">
        <p>{{ error }}</p>
      </div>

      <template v-else-if="list">
        <!-- Shared list banner -->
        <div class="sb-banner">
          <div class="sb-banner-arrow"></div>
          <span>{{ list.ownerName }} has shared a list with you &mdash; {{ items.length }} {{ items.length === 1 ? 'product' : 'products' }}</span>
        </div>

        <h1 class="sb-heading">{{ list.name }}</h1>

        <!-- View toggle -->
        <div v-if="items.length > 0" class="sb-view-toggle">
          <button
            class="sb-toggle-btn" :class="{ active: listView === 'grid' }"
            @click="listView = 'grid'" title="Rutn&auml;t"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><rect x="0" y="0" width="7" height="7" rx="1"/><rect x="9" y="0" width="7" height="7" rx="1"/><rect x="0" y="9" width="7" height="7" rx="1"/><rect x="9" y="9" width="7" height="7" rx="1"/></svg>
          </button>
          <button
            class="sb-toggle-btn" :class="{ active: listView === 'list' }"
            @click="listView = 'list'" title="Lista"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><rect x="0" y="1" width="16" height="2" rx="1"/><rect x="0" y="7" width="16" height="2" rx="1"/><rect x="0" y="13" width="16" height="2" rx="1"/></svg>
          </button>
        </div>

        <!-- Grid view -->
        <div v-if="listView === 'grid'" class="sb-grid">
          <a
            v-for="item in items" :key="item.productId"
            :href="productUrl(item)"
            target="_blank"
            rel="noopener"
            class="sb-card"
          >
            <div class="sb-card-img">
              <img :src="imageUrl(item, '200')" :alt="item.productNameBold" loading="lazy" />
            </div>
            <div class="sb-card-body">
              <div class="sb-card-badges">
                <span v-if="item.isOrganic" class="sb-badge sb-badge-eco">Ekologisk</span>
              </div>
              <p class="sb-card-category">{{ item.categoryLevel1 }} &middot; {{ item.categoryLevel2 }}</p>
              <h3 class="sb-card-name">
                {{ item.productNameBold }}
                <span v-if="item.productNameThin" class="sb-card-name-thin"> {{ item.productNameThin }}</span>
              </h3>
              <p class="sb-card-producer">{{ item.producerName }}</p>
              <div class="sb-card-meta">
                <span>{{ item.price.toFixed(2) }} kr</span>
                <span class="sb-dot"></span>
                <span>{{ item.volumeText }}</span>
                <span class="sb-dot"></span>
                <span>{{ item.alcoholPercentage }}%</span>
              </div>
              <p v-if="item.taste" class="sb-card-taste">{{ tastePreview(item) }}</p>
              <p class="sb-card-country">{{ item.country }}</p>
            </div>
          </a>
        </div>

        <!-- List view -->
        <div v-else class="sb-list">
          <a
            v-for="item in items" :key="item.productId"
            :href="productUrl(item)"
            target="_blank"
            rel="noopener"
            class="sb-list-item"
          >
            <div class="sb-list-img">
              <img :src="imageUrl(item, '100')" :alt="item.productNameBold" loading="lazy" />
            </div>
            <div class="sb-list-body">
              <p class="sb-list-category">{{ item.categoryLevel1 }} &middot; {{ item.categoryLevel2 }}</p>
              <h3 class="sb-list-name">
                {{ item.productNameBold }}
                <span v-if="item.productNameThin" class="sb-card-name-thin"> {{ item.productNameThin }}</span>
              </h3>
              <p class="sb-list-producer">{{ item.producerName }}</p>
            </div>
            <div class="sb-list-right">
              <span class="sb-list-price">{{ item.price.toFixed(2) }} kr</span>
              <span class="sb-list-vol">{{ item.volumeText }} &middot; {{ item.alcoholPercentage }}%</span>
              <span class="sb-list-country">{{ item.country }}</span>
            </div>
          </a>
        </div>

      </template>
    </main>

  </div>
</template>

<style scoped>
/* ── Systembolaget-inspired design ── */

.sb-page {
  font-family: 'Outfit', system-ui, -apple-system, sans-serif;
  background: #ffffff;
  min-height: 100vh;
  color: #2D2926;
  margin: 0;
  padding: 0;
  position: relative;
  overflow-x: hidden;
}

/* Header */
.sb-header {
  border-bottom: 1px solid #e5e5e5;
  background: #ffffff;
}

.sb-header-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem 1.5rem;
  display: flex;
  align-items: center;
}

.sb-brand {
  font-size: 1.2rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: #2d6a4f;
}

/* Main content */
.sb-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1.5rem;
}

/* Loading */
.sb-loading {
  display: flex;
  justify-content: center;
  padding: 4rem 0;
}

.sb-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e5e5e5;
  border-top-color: #2D2926;
  border-radius: 50%;
  animation: sb-spin 0.8s linear infinite;
}

@keyframes sb-spin {
  to { transform: rotate(360deg); }
}

/* Error */
.sb-error {
  text-align: center;
  padding: 4rem 0;
  color: #666;
}

/* Banner */
.sb-banner {
  position: relative;
  background: #f5f0eb;
  padding: 0.75rem 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
  font-size: 0.9rem;
  color: #2D2926;
}

.sb-banner-arrow {
  position: absolute;
  bottom: -6px;
  left: 1.5rem;
  width: 12px;
  height: 12px;
  background: #f5f0eb;
  transform: rotate(45deg);
}

/* Heading */
.sb-heading {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0 0 1rem 0;
  letter-spacing: -0.02em;
}

/* View toggle */
.sb-view-toggle {
  display: flex;
  justify-content: flex-end;
  gap: 0.25rem;
  margin-bottom: 1rem;
}

.sb-toggle-btn {
  border: 1px solid #d4d4d4;
  background: #fff;
  padding: 0.4rem;
  border-radius: 4px;
  cursor: pointer;
  color: #888;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sb-toggle-btn:hover {
  border-color: #999;
  color: #2D2926;
}

.sb-toggle-btn.active {
  background: #2D2926;
  border-color: #2D2926;
  color: #fff;
}

/* Grid */
.sb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.sb-card {
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  text-decoration: none;
  color: inherit;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.sb-card:hover {
  border-color: #ccc;
  box-shadow: 0 4px 16px rgba(0,0,0,0.06);
}

.sb-card-img {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 1.5rem;
  background: #fafafa;
  min-height: 180px;
}

.sb-card-img img {
  max-height: 150px;
  max-width: 100%;
  object-fit: contain;
}

.sb-card-body {
  padding: 1rem;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.sb-card-badges {
  display: flex;
  gap: 0.35rem;
  margin-bottom: 0.25rem;
}

.sb-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 12px;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.sb-badge-eco {
  background: #e8f5e9;
  color: #2e7d32;
}

.sb-card-category {
  font-size: 0.75rem;
  color: #888;
  margin: 0 0 0.25rem 0;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.sb-card-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.15rem 0;
  line-height: 1.3;
}

.sb-card-name-thin {
  font-weight: 400;
  color: #666;
}

.sb-card-producer {
  font-size: 0.85rem;
  color: #666;
  margin: 0 0 0.5rem 0;
}

.sb-card-meta {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  font-weight: 500;
  margin-top: auto;
}

.sb-dot {
  width: 3px;
  height: 3px;
  background: #bbb;
  border-radius: 50%;
}

.sb-card-taste {
  font-size: 0.8rem;
  color: #888;
  margin: 0.4rem 0 0 0;
  line-height: 1.4;
}

.sb-card-country {
  font-size: 0.75rem;
  color: #aaa;
  margin: 0.25rem 0 0 0;
}

/* List view */
.sb-list {
  display: flex;
  flex-direction: column;
  gap: 0;
  margin-bottom: 2rem;
}

.sb-list-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid #f0f0f0;
  text-decoration: none;
  color: inherit;
  transition: background 0.15s;
}

.sb-list-item:hover {
  background: #fafafa;
}

.sb-list-img {
  width: 48px;
  flex-shrink: 0;
  display: flex;
  justify-content: center;
}

.sb-list-img img {
  max-height: 48px;
  max-width: 48px;
  object-fit: contain;
}

.sb-list-body {
  flex: 1;
  min-width: 0;
}

.sb-list-category {
  font-size: 0.7rem;
  color: #888;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.sb-list-name {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0.1rem 0;
  line-height: 1.3;
}

.sb-list-producer {
  font-size: 0.8rem;
  color: #666;
  margin: 0;
}

.sb-list-right {
  text-align: right;
  flex-shrink: 0;
}

.sb-list-price {
  display: block;
  font-weight: 600;
  font-size: 0.95rem;
}

.sb-list-vol {
  display: block;
  font-size: 0.75rem;
  color: #888;
}

.sb-list-country {
  display: block;
  font-size: 0.7rem;
  color: #aaa;
}


@media (max-width: 768px) {
  .sb-header-inner {
    padding: 0.75rem 1rem;
  }

  .sb-brand {
    font-size: 1rem;
  }

  .sb-main {
    padding: 1rem;
  }

  .sb-heading {
    font-size: 1.25rem;
  }

  .sb-banner {
    font-size: 0.82rem;
    padding: 0.6rem 0.75rem;
  }

  .sb-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .sb-card-img {
    min-height: 140px;
    padding: 1rem;
  }

  .sb-card-img img {
    max-height: 120px;
  }

  .sb-list-item {
    gap: 0.75rem;
    padding: 0.6rem 0;
  }

  .sb-list-body {
    min-width: 0;
  }

  .sb-list-name {
    font-size: 0.88rem;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sb-list-right {
    min-width: 70px;
  }
}

@media (max-width: 400px) {
  .sb-main {
    padding: 0.75rem;
  }

  .sb-card-body {
    padding: 0.75rem;
  }

  .sb-card-meta {
    flex-wrap: wrap;
    gap: 0.25rem;
  }

  .sb-toggle-btn {
    padding: 0.5rem;
  }
}
</style>
