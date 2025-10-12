<template>
  <v-form>
    <v-row>
      <v-col
          v-for="(f, idx) in step.fields"
          :key="idx"
          cols="12"
          md="6"
      >
        <!-- Role -->
        <v-select
            v-if="isConfigureServer && idx === 0"
            v-model="f.value"
            :items="[
    { title: 'control-plane', value: '1' },
    { title: 'worker', value: '2' },
  ]"
            label="Select Role"
        />

        <!-- Disk -->
        <v-select
            v-else-if="isConfigureServer && idx === 1"
            v-model="f.value"
            :items="diskOptions"
            item-title="label"
            item-value="value"
            label="Select Disk"
        />


        <!-- Default field type -->
        <template v-else>
          <v-text-field
              v-model="f.value"
              :label="f.label"
              :placeholder="f.placeholder"
          />
        </template>
      </v-col>
    </v-row>
  </v-form>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'

const props = defineProps({ step: Object })
const diskOptions = ref([])

const isConfigureServer = computed(() =>
    props.step.name.startsWith('ConfigureServer_')
)

async function loadDisks() {
  if (!isConfigureServer.value) return
  const ip = props.step.name.replace('ConfigureServer_', '')
  const { data } = await axios.get(`/api/nodes/${ip}/disks`)
  diskOptions.value = data.map((d, i) => ({
    label: `${i + 1}) ${d.name} (${d.model})`,
    value: String(i + 1), // 👈 send numeric string expected by TUI
  }))
}

onMounted(loadDisks)

</script>
