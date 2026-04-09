<script setup lang="ts">
import ProductTable from './components/ProductTable.vue'
import SyncPanel from './components/SyncPanel.vue'
import ApiKeyStatus from './components/ApiKeyStatus.vue'
import BasketPanel from './components/BasketPanel.vue'
import EventsPage from './components/EventsPage.vue'
import SharedListsPage from './components/SharedListsPage.vue'
import SharedListView from './components/SharedListView.vue'
import LoginForm from './components/LoginForm.vue'
import AdminPanel from './components/AdminPanel.vue'
import ChangePassword from './components/ChangePassword.vue'
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useAuthStore } from './stores/auth'
import type { AuthUser } from './api/client'

const authStore = useAuthStore()

// Check if this is a public shared list page
const sharedListUUID = ref<string | null>(null)
const pathMatch = window.location.pathname.match(/^\/delad-lista\/([a-f0-9-]+)$/)
if (pathMatch) {
  sharedListUUID.value = pathMatch[1]
}

const showSync = ref(false)
const showBaskets = ref(false)
const showEvents = ref(false)
const showLists = ref(false)
const showAdmin = ref(false)
const showSettings = ref(false)
const tableRef = ref<InstanceType<typeof ProductTable>>()
const basketRef = ref<InstanceType<typeof BasketPanel>>()
const sharedListRef = ref<InstanceType<typeof SharedListsPage>>()
const activeBasketId = ref<number | undefined>()
const activeSharedListId = ref<number | undefined>()

// Impersonate dropdown
const users = ref<AuthUser[]>([])
const showUserMenu = ref(false)
const userMenuRef = ref<HTMLElement>()

const isRealAdmin = computed(() =>
  authStore.isAdmin || authStore.isImpersonating
)

const impersonateUsers = computed(() => users.value)

async function loadUsers() {
  if (!isRealAdmin.value) return
  try {
    // When impersonating, use admin token to fetch users
    const tkn = authStore.isImpersonating
      ? localStorage.getItem('adminToken')
      : localStorage.getItem('token')
    const res = await fetch('/api/admin/users', {
      headers: { Authorization: `Bearer ${tkn}` },
    })
    if (res.ok) {
      users.value = await res.json()
    }
  } catch {
    // not admin or error, ignore
  }
}

async function handleImpersonate(userId: number) {
  showUserMenu.value = false
  try {
    // If already impersonating, restore admin first so the token is correct
    if (authStore.isImpersonating) {
      authStore.stopImpersonating()
    }
    await authStore.impersonate(userId)
  } catch (e: any) {
    console.error('Impersonation failed:', e)
  }
}

function onClickOutside(e: MouseEvent) {
  if (userMenuRef.value && !userMenuRef.value.contains(e.target as Node)) {
    showUserMenu.value = false
  }
}

onMounted(() => {
  loadUsers()
  document.addEventListener('click', onClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
})

function onSynced() {
  showSync.value = false
  tableRef.value?.reload()
  basketRef.value?.refreshActive()
  sharedListRef.value?.refreshActive()
}

function onActiveBasketChanged(id: number | undefined) {
  activeBasketId.value = id
}

function onBasketChanged() {
  basketRef.value?.refreshActive()
}

function onNewBasket(id: number) {
  activeBasketId.value = id
  showBaskets.value = true
  basketRef.value?.refreshActive()
}
</script>

<template>
  <!-- Public shared list page (no auth needed) -->
  <SharedListView v-if="sharedListUUID" :uuid="sharedListUUID" />

  <template v-else-if="authStore.isLoggedIn">
    <div v-if="authStore.isImpersonating" class="impersonation-banner">
      Viewing as <strong>{{ authStore.user?.username }}</strong>
      <button class="impersonation-stop" @click="authStore.stopImpersonating()">
        Stop Impersonating
      </button>
    </div>

    <nav class="navbar">
      <h1>Systemet</h1>
      <button class="nav-btn" :class="{ active: showBaskets && !showEvents && !showLists }" @click="showBaskets = !showBaskets; showEvents = false; showLists = false">
        <i class="pi pi-shopping-cart"></i> Baskets
      </button>
      <button class="nav-btn" :class="{ active: showEvents }" @click="showEvents = !showEvents; if (showEvents) { showBaskets = false; showLists = false }">
        <i class="pi pi-calendar"></i> Events
      </button>
      <button class="nav-btn" :class="{ active: showLists }" @click="showLists = !showLists; if (showLists) { showBaskets = false; showEvents = false }">
        <i class="pi pi-list"></i> Lists
      </button>
      <button class="nav-btn" :class="{ active: showSync }" @click="showSync = !showSync">
        <i class="pi pi-sync"></i> Sync
      </button>
      <button class="nav-btn" :class="{ active: showSettings }" @click="showSettings = !showSettings">
        <i class="pi pi-cog"></i> Settings
      </button>

      <!-- Impersonate dropdown -->
      <div v-if="isRealAdmin" class="user-menu-wrapper" ref="userMenuRef">
        <button class="nav-btn" :class="{ active: showUserMenu }" @click.stop="showUserMenu = !showUserMenu; loadUsers()">
          <i class="pi pi-eye"></i> View as
        </button>
        <div v-if="showUserMenu" class="user-menu">
          <div class="user-menu-header">Switch user view</div>
          <div
            v-for="u in impersonateUsers" :key="u.id"
            class="user-menu-item"
            @click="handleImpersonate(u.id)"
          >
            <i class="pi pi-user"></i>
            <span>{{ u.username }}</span>
            <span class="user-role" :class="u.role">{{ u.role }}</span>
          </div>
          <div v-if="impersonateUsers.length === 0" class="user-menu-empty">
            No other users
          </div>
        </div>
      </div>
      <button v-if="isRealAdmin" class="nav-btn" :class="{ active: showAdmin }" @click="showAdmin = !showAdmin">
        <i class="pi pi-users"></i> Admin
      </button>

      <div class="nav-spacer"></div>
      <ApiKeyStatus />
      <span class="nav-user">
        <i class="pi pi-user"></i> {{ authStore.user?.username }}
      </span>
      <button class="nav-btn-logout" @click="authStore.logout()">
        <i class="pi pi-sign-out"></i> Logout
      </button>
    </nav>

    <AdminPanel v-if="showAdmin" @close="showAdmin = false" @productsChanged="tableRef?.reload(); basketRef?.refreshActive(); sharedListRef?.refreshActive()" />
    <ChangePassword v-if="showSettings" @close="showSettings = false" />
    <SyncPanel v-if="showSync" @synced="onSynced" @cancel="showSync = false" />

    <template v-if="showEvents">
      <EventsPage />
    </template>
    <template v-else>
      <SharedListsPage v-if="showLists" ref="sharedListRef" @update:activeId="(id: number | undefined) => activeSharedListId = id" />
      <BasketPanel v-if="showBaskets && !showLists" ref="basketRef" @update:activeId="onActiveBasketChanged" />
      <ProductTable ref="tableRef" :activeBasketId="activeBasketId" :activeSharedListId="activeSharedListId" @basketChanged="onBasketChanged" @newBasket="onNewBasket" @sharedListChanged="sharedListRef?.refreshActive()" />
    </template>
  </template>

  <LoginForm v-else />
</template>

<style scoped>
.user-menu-wrapper {
  position: relative;
}

.user-menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 0.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  min-width: 200px;
  z-index: 100;
  overflow: hidden;
}

.user-menu-header {
  padding: 0.5rem 0.75rem;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border-light);
}

.user-menu-item {
  padding: 0.5rem 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.85rem;
  color: var(--text-secondary);
  transition: background 0.15s;
}

.user-menu-item:hover {
  background: var(--bg-muted);
}

.user-menu-item i {
  font-size: 0.8rem;
  color: var(--text-faint);
}

.user-role {
  margin-left: auto;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 10px;
}

.user-role.admin {
  background: var(--purple-light);
  color: var(--purple);
}

.user-role.user {
  background: var(--accent-light);
  color: var(--accent);
}

.user-menu-empty {
  padding: 0.75rem;
  color: var(--text-faint);
  font-size: 0.8rem;
  text-align: center;
}
</style>
