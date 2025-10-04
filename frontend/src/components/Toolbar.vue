<template>
    <v-app-bar :height="props.toolbarHeight" app color="primary" dark :elevation="props.elevation">
      <v-btn icon @click="toggleSidebar">
        <v-icon>mdi-menu</v-icon>
      </v-btn>
      <v-spacer></v-spacer>
    </v-app-bar>
</template>

<script setup>
// Props
const props = defineProps({
  drawer: {
    type: Boolean,
    required: true,
  },
  toolbarHeight: {
    type: Number,
    default: 64,
  },
  elevation: {
    type: Number,
    default: 5,
  },
});

// Emits
const emit = defineEmits(['update:drawer']);

// Methods
function toggleSidebar() {
  emit('update:drawer', !props.drawer);
}
</script>