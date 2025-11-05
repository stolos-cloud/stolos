<template>
    <v-main>
        <Toolbar 
            v-if="!isDesktop"
            :drawer="drawer" 
            :toolbarHeight="toolbarHeight" 
            @update:drawer="drawer = $event" 
        ></Toolbar>
        <SidebarUI 
            :drawer="drawer" 
            :toolbarHeight="toolbarHeight" 
            :temporary="!isDesktop" 
            @update:drawer="drawer = $event"
        />
        <v-container fluid>
            <slot></slot>
        </v-container>
    </v-main>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useDisplay } from 'vuetify';

const drawer = ref(false);
const toolbarHeight = ref(50);
const  { mdAndUp } = useDisplay();

const isDesktop = computed(() => mdAndUp.value);
</script>
