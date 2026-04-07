import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { definePreset } from '@primevue/themes'
import Aura from '@primevue/themes/aura'
import 'primeicons/primeicons.css'
import './style.css'
import App from './App.vue'

const SystemetPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '#e8f5ee',
      100: '#d0ebdb',
      200: '#a5d6b9',
      300: '#74c096',
      400: '#4aab79',
      500: '#2d6a4f',
      600: '#256243',
      700: '#1b4332',
      800: '#143326',
      900: '#0e241a',
      950: '#091610',
    },
  },
})

const app = createApp(App)

app.use(createPinia())
app.use(PrimeVue, {
  theme: {
    preset: SystemetPreset,
  },
})

app.mount('#app')
