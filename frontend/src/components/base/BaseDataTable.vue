<template>
    <div>
        <v-sheet class="mt-4 border rounded-lg">
            <v-data-table 
                :headers="headers" 
                :items="items" 
                :items-length="items.length" 
                :loading="loading"
                :search="internalSearch" 
                :loading-text="loadingText"
                :no-data-text="noDataText" 
                :items-per-page="10"
                :items-per-page-options="[10]"
                :items-per-page-text="itemsPerPageText" 
                :hide-default-footer="items.length < 10"
                :height="height"
                :fixed-header="fixedHeader"
                mobile-breakpoint="md"
                v-on="rowClickable ? { 'click:row': handleClickRow } : {}"
                :class="{ 'clickable-rows': rowClickable}"
            >
                <template v-slot:top>
                    <BaseToolbarTable 
                        v-model="internalSearch" 
                        :title="titleToolbar" 
                        :buttons="actionsButtonForTable" 
                    />
                </template>
                <template 
                    v-for="name in Object.keys($slots)"
                    v-slot:[name]="slotProps"
                >
                    <slot :name="name" v-bind="slotProps"></slot>
                </template>
            </v-data-table>
        </v-sheet>
        <div 
            v-if="footerMessage"
            class="text-right py-1 text-caption text-grey">
            {{ footerMessage }}
        </div>
    </div>
</template>

<script setup>
import { ref, watch } from "vue";

const props = defineProps({
    modelValue: {
        type: String,
        default: undefined
    },
    headers: {
        type: Array,
        required: true
    },
    items: {
        type: Array,
        required: true
    },
    loading: {
        type: Boolean,
        default: false
    },
    loadingText: {
        type: String,
        default: ''
    },
    noDataText: {
        type: String,
        default: ''
    },
    itemsPerPageText: {
        type: String,
        default: ''
    },
    titleToolbar: {
        type: String,
        default: ''
    },
    actionsButtonForTable: {
        type: Array,
        default: () => []
    },
    rowClickable: {
        type: Boolean,
        default: false
    },
    footerMessage: {
        type: String,
        default: ''
    },
    height: {
        type: [String, Number],
        default: null
    },
    fixedHeader: {
        type: Boolean,
        default: false
    }
});

// State
const internalSearch = ref(props.modelValue);

// Emits
const emits = defineEmits(['update:modelValue', 'click:row']);

// Watchers
watch(internalSearch, (val) => emits('update:modelValue', val));
watch(() => props.modelValue, (val) => {
    if (val !== internalSearch.value) internalSearch.value = val;
});

// Methods
function handleClickRow(event, item) {
    if (!props.rowClickable) return;
    if (event.target.closest('.v-btn, .v-icon')) return;
    emits('click:row', event, item);
}
</script>

<style scoped>
:deep(.clickable-rows .v-data-table__tr:hover) {
  background-color: rgba(var(--v-theme-on-surface), 0.1);
}
</style>