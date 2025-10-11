<template>
  <v-card class="mt-6 pa-2" outlined max-height="300" style="overflow:auto;">
    <v-card-title class="text-subtitle-1">Logs</v-card-title>
    <v-card-text>
      <pre style="white-space: pre-wrap; font-family: monospace;">
{{ logs.join('\n') }}
      </pre>
    </v-card-text>
  </v-card>
</template>

<script setup>
defineProps({ logs: Array })
</script>
