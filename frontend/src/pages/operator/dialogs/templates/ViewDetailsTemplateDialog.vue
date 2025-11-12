<template>
    <div class="view-details-template-dialog">
        <BaseDialog v-model="isOpen" title="$t('templateDefinitions.dialogs.viewDetailsTemplate.title')" closable>
            <v-expansion-panels class="my-4">
                <v-expansion-panel class="border">
                    <v-expansion-panel-title class="px-3">
                        <BaseTitle :level="6"
                            :title="$t('templateDefinitions.dialogs.viewDetailsTemplate.yamlTitle')" />
                        <v-chip color="primary" size="small" label class="ml-2">
                            v{{ template?.version || '1.0' }}
                        </v-chip>
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                        <v-sheet color="grey-darken-4" rounded>
                            <pre class="yaml-block">
                                    {{ yamlContent }}
                                </pre>
                        </v-sheet>
                    </v-expansion-panel-text>
                </v-expansion-panel>
            </v-expansion-panels>
            <v-card class="border my-4">
                <v-card-title class="my-2">
                    <div class="d-flex align-center">
                        <BaseTitle :level="6" :title="$t('templateDefinitions.dialogs.viewDetailsTemplate.footerTitle')" />
                        <v-chip size="small"  
                            label class="ml-2">
                            2 total
                        </v-chip>
                        <v-spacer />
                        <v-btn                      
                            variant="text"        
                            icon="mdi-open-in-new" 
                            color="primary"
                            size="small">
                        </v-btn>
                    </div>
                </v-card-title>
                <v-divider></v-divider>
                <v-card-text>
                    <v-virtual-scroll
                        :items="templates"
                        max-height="100"
                        v-if="templates.length > 0"
                    >
                        <template v-slot:default="{ item }">
                            <v-list lines="two">
                                <v-list-item
                                    :key="item.id"
                                    :title="item.email"
                                    class="border rounded"
                                    style="background-color: rgba(33, 33, 33);"
                                >
                                    <template #subtitle>
                                        <div class="d-flex align-center">
                                            <span class="text-caption text-medium-emphasis">{{ item.id }}</span>
                                            <v-btn
                                                class="ml-1"
                                                :icon="copiedItem === item.id ? 'mdi-check' : 'mdi-content-copy'"
                                                size="x-small"
                                                variant="text"
                                                @click="copyToClipboard(item.id)"
                                            />
                                        </div>
                                    </template>
                                    <template v-slot:append>
                                        <v-chip size="small"  
                                            color="success"
                                            label>
                                            running
                                        </v-chip>
                                    </template>
                                </v-list-item>
                            </v-list>
                        </template>
                    </v-virtual-scroll>
                </v-card-text>


            </v-card>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, watch } from "vue";

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    template: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);


// Emits
const emit = defineEmits(['update:modelValue']);
const templates = ref([
    { id: 'template-1', email: 'user1@example.com' },
    { id: 'template-2', email: 'user2@example.com' },
    { id: 'template-3', email: 'user3@example.com' }
]);
const yamlContent = ref({
    "name": "example-template",
    "version": "1.0",
    "description": "An example template configuration",
    "components": {
        "frontend": {
            "image": "example/frontend:latest",
            "replicas": 2,
            "resources": {
                "limits": {
                    "cpu": "500m",
                    "memory": "256Mi"
                },
                "requests": {
                    "cpu": "250m",
                    "memory": "128Mi"
                }
            }
        },
        "backend": {
            "image": "example/backend:latest",
            "replicas": 3,
            "resources": {
                "limits": {
                    "cpu": "1",
                    "memory": "512Mi"
                },
                "requests": {
                    "cpu": "500m",
                    "memory": "256Mi"
                }
            }
        }
    }
});

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods

</script>

<style>
.v-expansion-panel--active>.v-expansion-panel-title:not(.v-expansion-panel-title--static) {
    border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.1) !important;
}

.v-expansion-panel-text__wrapper {
    padding: 10px 12px 10px !important;
}
</style>