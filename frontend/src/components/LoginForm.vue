<script setup lang="ts">
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function handleLogin() {
  loading.value = true
  error.value = null
  try {
    await authStore.login(username.value, password.value)
  } catch (e: any) {
    error.value = e.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-overlay">
    <div class="login-card">
      <h1>Systemet</h1>
      <p class="login-sub">Sign in to continue</p>
      <form @submit.prevent="handleLogin" class="login-form">
        <div class="field">
          <label>Username</label>
          <InputText v-model="username" placeholder="Username" autofocus style="width: 100%;" />
        </div>
        <div class="field">
          <label>Password</label>
          <Password v-model="password" :feedback="false" toggleMask placeholder="Password"
            :inputStyle="{ width: '100%' }" style="width: 100%;" />
        </div>
        <div v-if="error" class="login-error">{{ error }}</div>
        <Button type="submit" label="Sign In" icon="pi pi-sign-in" :loading="loading" style="width: 100%;" />
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-overlay {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
}
.login-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  padding: 2.5rem;
  width: 100%;
  max-width: 380px;
  box-shadow: var(--shadow-md);
}
.login-card h1 {
  margin: 0 0 0.25rem;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent);
  letter-spacing: -0.03em;
}
.login-sub {
  color: var(--text-muted);
  margin: 0 0 1.5rem;
  font-size: 0.9rem;
}
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.login-error {
  color: var(--danger);
  font-size: 0.85rem;
  background: var(--danger-light);
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
}
</style>
