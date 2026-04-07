<script setup lang="ts">
import { ref, computed } from 'vue'
import Password from 'primevue/password'
import Button from 'primevue/button'
import { changePassword } from '../api/client'

const emit = defineEmits<{ close: [] }>()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref<string | null>(null)
const success = ref(false)
const loading = ref(false)

const hasMinLength = computed(() => newPassword.value.length >= 10)
const hasUpper = computed(() => /[A-Z]/.test(newPassword.value))
const hasDigit = computed(() => /\d/.test(newPassword.value))
const passwordsMatch = computed(() => newPassword.value !== '' && newPassword.value === confirmPassword.value)
const canSubmit = computed(() => currentPassword.value && hasMinLength.value && hasUpper.value && hasDigit.value && passwordsMatch.value)

async function doChange() {
  loading.value = true
  error.value = null
  success.value = false
  try {
    await changePassword(currentPassword.value, newPassword.value)
    success.value = true
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    setTimeout(() => emit('close'), 1500)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h3>Change Password</h3>
      <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="$emit('close')" />
    </div>

    <form @submit.prevent="doChange" class="pw-form">
      <div class="field">
        <label>Current password</label>
        <Password v-model="currentPassword" :feedback="false" toggleMask :inputStyle="{ width: '100%' }" style="width: 100%;" />
      </div>
      <div class="field">
        <label>New password</label>
        <Password v-model="newPassword" :feedback="false" toggleMask :inputStyle="{ width: '100%' }" style="width: 100%;" />
      </div>
      <div class="field">
        <label>Confirm new password</label>
        <Password v-model="confirmPassword" :feedback="false" toggleMask :inputStyle="{ width: '100%' }" style="width: 100%;" />
      </div>

      <div class="pw-rules">
        <span :class="{ met: hasMinLength }">
          <i :class="hasMinLength ? 'pi pi-check' : 'pi pi-circle'" style="font-size: 0.65rem;"></i>
          At least 10 characters
        </span>
        <span :class="{ met: hasUpper }">
          <i :class="hasUpper ? 'pi pi-check' : 'pi pi-circle'" style="font-size: 0.65rem;"></i>
          At least one uppercase letter
        </span>
        <span :class="{ met: hasDigit }">
          <i :class="hasDigit ? 'pi pi-check' : 'pi pi-circle'" style="font-size: 0.65rem;"></i>
          At least one number
        </span>
        <span :class="{ met: passwordsMatch }">
          <i :class="passwordsMatch ? 'pi pi-check' : 'pi pi-circle'" style="font-size: 0.65rem;"></i>
          Passwords match
        </span>
      </div>

      <div v-if="error" class="pw-error">{{ error }}</div>
      <div v-if="success" class="pw-success">Password changed successfully!</div>

      <Button type="submit" label="Change Password" :loading="loading" :disabled="!canSubmit" style="align-self: flex-start;" />
    </form>
  </div>
</template>

<style scoped>
.pw-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 360px;
}

.pw-rules {
  font-size: 0.8rem;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.pw-rules span {
  color: var(--text-faint);
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.pw-rules span.met {
  color: var(--success);
}

.pw-error {
  color: var(--danger);
  font-size: 0.85rem;
  background: var(--danger-light);
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
}

.pw-success {
  color: var(--success);
  font-weight: 600;
  font-size: 0.875rem;
}
</style>
