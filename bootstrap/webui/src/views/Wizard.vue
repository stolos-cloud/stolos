<template>
  <v-container>
    <VStepper v-model="activeStep" non-linear class="mb-6">
      <VStepperHeader>
        <template v-for="(s, i) in steps" :key="s.name">
          <VStepperItem
              :value="i + 1"
              :title="s.title"
              :complete="s.isDone"
          />
          <VDivider v-if="i < steps.length - 1" />
        </template>
      </VStepperHeader>

      <VStepperWindow>
        <template v-for="(s, i) in steps" :key="s.name">
          <VStepperWindowItem :value="i + 1">
            <v-card outlined class="pa-4">
              <component :is="stepComponent(s)" :step="s" />
              <v-divider class="my-4" />
              <div class="d-flex justify-end">
                <v-btn
                    color="primary"
                    :disabled="!s.isDone || i === steps.length - 1"
                    @click="nextStep(s)"
                >
                  Next
                </v-btn>
              </div>
            </v-card>
          </VStepperWindowItem>
        </template>
      </VStepperWindow>
    </VStepper>

    <LogsConsole :logs="logs" />
  </v-container>
</template>

<script setup>
import {ref, onMounted, watch, h} from 'vue'
import axios from 'axios'
import LogsConsole from '../components/LogsConsole.vue'
import StepForm from '../components/StepForm.vue'

const steps = ref([])
const logs = ref([])
const activeStep = ref(1)

async function fetchSteps() {
  const { data } = await axios.get('/api/steps')
  steps.value = Array.isArray(data) ? data : []
}

async function fetchLogs() {
  try {
    const { data } = await axios.get('/api/logs')
    if (!Array.isArray(data)) {
      logs.value = []
      return
    }
    logs.value = data.map(l => `[${l.At || ''}] ${l.Text || ''}`)
  } catch {
    logs.value = []
  }
}

async function fetchCurrentStep() {
  try {
    const { data } = await axios.get('/api/currentstep')
    if (typeof data.index === 'number' && data.index >= 0) {
      activeStep.value = data.index + 1

      const idx = data.index
      if (steps.value[idx]) {
        // ✅ Only update mutable runtime state
        steps.value[idx].body = data.body
        steps.value[idx].isDone = data.isDone
      }
    }
  } catch (e) {
    console.warn('Failed to fetch current step', e)
  }
}



function stepComponent(step) {
  switch (step.kind) {
    case 0:
      // Form step
      return StepForm

    case 1:
      // Spinner step
      return {
        props: ['step'],
        render() {
          return h('div', { class: 'text-center py-6' }, [
            h('v-progress-circular', {
              indeterminate: true,
              color: 'primary',
              size: 32,
            }),
            h('div', { class: 'mt-3 text-subtitle-1' }, this.step.body || 'Working...')
          ])
        },
      }

    default:
      // Plain step
      return {
        props: ['step'],
        render() {
          return h('div', { class: 'py-4' }, this.step.body)
        },
      }
  }
}



async function nextStep(step) {
  const payload = { fields: step.fields || [] }
  await axios.post(`/api/steps/${step.name}/next`, payload)
  await fetchSteps()
  await fetchCurrentStep()
}

onMounted(async () => {
  await fetchSteps()
  await fetchLogs()
  await fetchCurrentStep()

  // Poll logs & current step every 2s
  setInterval(fetchLogs, 2000)
  setInterval(fetchCurrentStep, 2000)
})
</script>
