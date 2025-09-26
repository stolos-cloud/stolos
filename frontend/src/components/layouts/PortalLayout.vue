<template>
  <v-app>
    <Sidebar :drawer="drawer" :toolbarHeight="toolbarHeight"></Sidebar>
    <Toolbar 
      :drawer="drawer" 
      :toolbarHeight="toolbarHeight" 
      @update:drawer="drawer = $event" 
      @go-to-profile="goToProfile" 
    ></Toolbar>

    <v-main>
      <v-container fluid>
        <slot></slot>
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import { ref } from 'vue';

const drawer = ref(true);
const toolbarHeight = ref(50);
</script>
