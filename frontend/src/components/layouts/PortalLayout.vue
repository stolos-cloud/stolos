<template>
  <v-app>
    <Toolbar 
      v-if="!isDesktop"
      :drawer="drawer" 
      :toolbarHeight="toolbarHeight" 
      @update:drawer="drawer = $event" 
    ></Toolbar>
    <Sidebar 
      :drawer="drawer" 
      :toolbarHeight="toolbarHeight" 
      :temporary="!isDesktop" 
      @update:drawer="drawer = $event"
    />
    <v-main>
      <v-container fluid>
        <slot></slot>
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useDisplay } from 'vuetify';

const drawer = ref(true);
const toolbarHeight = ref(50);
const  { mdAndUp } = useDisplay();

const isDesktop = computed(() => mdAndUp.value);
</script>
