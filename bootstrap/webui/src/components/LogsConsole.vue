<template>
  <v-sheet class="mt-6 pa-2" outlined max-height="25vh" style="overflow:auto;">
      <pre style="white-space: pre-wrap; font-family: monospace;">
{{ logs.join('\n') }}
      </pre>
  </v-sheet>
</template>

<script setup>
defineProps({ logs: Array })
</script>
