<template>
  <v-container fluid class="pa-0">
    <!-- Logs Console (fixed at top, overlay when expanded) -->
    <div class="logs-container" :class="{ expanded: logsExpanded }">
      <v-expansion-panels
          v-model="logsExpanded"
          flat
          class="logs-expander"
      >
        <v-expansion-panel>
          <v-expansion-panel-title>
            <div class="d-flex align-center justify-space-between w-100">
              <span class="text-h6">Logs Console</span>
              <v-icon>{{ logsExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
            </div>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <LogsConsole :logs="logs" :initialAutoScroll="true" />
          </v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>
    </div>

    <!-- Spacer so content starts below the fixed console -->
    <div style="height: 64px;"></div>

    <!-- Vertical Stepper -->
    <v-stepper-vertical v-model="activeStep" vertical class="px-4">
      <template v-for="(s, i) in steps" :key="s.name">
        <v-stepper-vertical-item
            :title="s.title"
            :value="i + 1"
            :complete="s.isDone"
            class="mb-4"
        >
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
        </v-stepper-vertical-item>
      </template>
    </v-stepper-vertical>
  </v-container>
</template>

<script setup>
import { ref, onMounted, h } from 'vue'
import { VStepperVertical, VStepperVerticalItem, VStepperVerticalActions } from 'vuetify/labs/VStepperVertical';
import axios from 'axios'
import LogsConsole from '../components/LogsConsole.vue'
import StepForm from '../components/StepForm.vue'

const steps = ref([])
const logs = ref([])
const activeStep = ref(1)
const logsExpanded = ref(false)

async function fetchSteps() {
  const { data } = await axios.get('/api/steps')
  steps.value = Array.isArray(data) ? data : []
}

// TODO : Send full log data to render log levels / move login to the LogConsole component
async function fetchLogs() {
  try {
    const { data } = await axios.get('/api/logs')
    logs.value = Array.isArray(data)
        ? data.map(l => `[${l.At || ''}] ${l.Text || ''}`)
        : []
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
      return StepForm
    case 1:
      return {
        props: ['step'],
        render() {
          return h('div', { class: 'text-center py-6' }, [
            h('v-progress-circular', {
              indeterminate: true,
              color: 'primary',
              size: 32,
            }),
            h('div', { class: 'mt-3 text-subtitle-1' }, this.step.body || 'Working...'),
          ])
        },
      }
    default:
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
  setInterval(fetchLogs, 2000)
  setInterval(fetchCurrentStep, 2000)
})
</script>

<style scoped>
.logs-container {
  position: fixed;
  top: 60px;
  left: 0;
  right: 0;
  z-index: 2000; /* above other content */
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
}

/* Expanded overlay mode */
.logs-container.expanded {
  height: 60vh; /* height of overlay */
  overflow: hidden;
}

.logs-container.expanded .v-expansion-panel-text {
  overflow-y: auto;
  height: calc(60vh - 56px); /* subtract title height */
}

.logs-expander {
  margin: 0;
}
</style>
