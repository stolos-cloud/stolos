<template>
  <v-form>
    <v-row>
      <v-col
          v-for="(f, idx) in step.fields"
          :key="idx"
          cols="12"
          md="6"
      >
        <v-text-field
            v-model="f.value"
            :label="f.label"
            :placeholder="f.placeholder"
        />
      </v-col>
    </v-row>
  </v-form>
</template>

<script setup>
defineProps({ step: Object })
</script>
