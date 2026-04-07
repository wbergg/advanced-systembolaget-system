<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import { listUsers, createUser, updateUser, deleteUser, type AuthUser } from '../api/client'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()
const emit = defineEmits<{ close: [] }>()

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

onMounted(loadUsers)
</script>

<template>
  <div class="card">
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
</style>
