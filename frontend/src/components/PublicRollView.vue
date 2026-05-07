<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RollGame from './RollGame.vue'
import { getPublicRoll, type PublicRollData } from '../api/client'

const data = ref<PublicRollData | null>(null)
const loading = ref(true)
const noActive = ref(false)

onMounted(async () => {
  document.body.style.background = '#ffffff'
  const body = document.querySelector('body') as HTMLElement
  if (body) body.classList.add('sb-public')

  try {
    data.value = await getPublicRoll()
  } catch {
    noActive.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="sb-page">
    <header class="sb-header">
      <div class="sb-header-inner">
        <span class="sb-brand">Advanced Systembolaget System</span>
      </div>
    </header>

    <main class="sb-main">
      <div v-if="loading" class="sb-loading">
        <div class="sb-spinner"></div>
      </div>

      <div v-else-if="noActive" class="sb-no-active">
        <p>Currently no active event.</p>
      </div>

      <template v-else-if="data">
        <h1 class="sb-heading">{{ data.eventName }}</h1>
        <div v-if="data.eventDate" class="sb-event-date">{{ data.eventDate }}</div>
        <p v-if="data.description" class="sb-event-desc">{{ data.description }}</p>
        <RollGame
          :eventId="0"
          :participants="data.participants"
          :isPublic="true"
        />
      </template>
    </main>
  </div>
</template>

<style scoped>
.sb-page {
  font-family: 'Outfit', system-ui, -apple-system, sans-serif;
  background: #ffffff;
  min-height: 100vh;
  color: #2D2926;
  margin: 0;
  padding: 0;
}

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

.sb-main {
  max-width: 700px;
  margin: 0 auto;
  padding: 1.5rem;
}

.sb-heading {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0 0 0.4rem 0;
  letter-spacing: -0.02em;
}

.sb-event-date {
  font-size: 0.95rem;
  color: #666;
  margin-bottom: 0.4rem;
}

.sb-event-desc {
  font-size: 0.95rem;
  color: #444;
  margin: 0 0 1rem 0;
  white-space: pre-wrap;
}

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

.sb-no-active {
  text-align: center;
  padding: 4rem 0;
  color: #666;
  font-size: 1.1rem;
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
}
</style>
