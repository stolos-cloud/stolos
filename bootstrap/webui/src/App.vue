<template>
  <v-app>
    <v-app-bar app color="primary" dark>
      <v-toolbar-title>Stolos Bootstrap Wizard</v-toolbar-title>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>

<script setup>
// nothing needed here yet
</script>

<style>
html, body, #app {
  height: 100%;
  margin: 0;
}
</style>
