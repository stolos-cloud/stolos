<template>
    <div class="view-details-namespace-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.namespaces.dialogs.viewDetailsNamespace.title', { namespaceName: namespaceName })" closable>
            <BaseCard>
                <template #title>
                    <BaseTitle :level="6" :title="$t('administration.namespaces.dialogs.viewDetailsNamespace.usersTitle')" />
                    <BaseChip class="ml-2">
                        {{ $t('administration.namespaces.dialogs.viewDetailsNamespace.totalMembersLabel', { count: usersNamespace.length }) }}
                    </BaseChip>
                </template>
                <v-virtual-scroll :items="usersNamespace" max-height="200" v-if="usersNamespace.length > 0">
                    <template v-slot:default="{ item }">
                        <v-list lines="two" style="background-color: rgba(var(--v-theme-grey));">
                            <v-list-item :key="item.id" :title="item.email" class="border rounded" style="background-color: rgba(var(--v-theme-list-item));">
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
                                    <v-btn
                                        v-tooltip="{ text: $t('administration.namespaces.buttons.deleteUserFromNamespace') }"
                                        icon="mdi-delete"
                                        size="small" variant="plain" color="primary"
                                        :disabled="currentUserId === item.id"
                                        @click="showConfirmDelete(item)"
                                    />
                                </template>
                            </v-list-item>
                        </v-list>
                    </template>
                </v-virtual-scroll>
                <div v-else class="no-members">
                    <p>{{ $t('administration.namespaces.dialogs.viewDetailsNamespace.messages.noMembers') }}</p>
                </div>
            </BaseCard>
        </BaseDialog>
        <BaseConfirmDialog ref="confirmDialog" />
    </div>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { getNamespaceDetails, deleteUserFromNamespaceByUserId } from "@/services/namespaces.service";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();
const store = useStore();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    namespace: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);
const usersNamespace = ref([]);
const confirmDialog = ref(null);
const copiedItem = ref(null);
const namespaceName = ref('');

// Emits
const emit = defineEmits(['update:modelValue', 'userDeletedFromNamespace']);

// Computed 
const currentUserId = computed(() => store.getters['user/getId']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if(val && props.namespace) {
        getNamespaceDetailsFromNamespaceId(props.namespace.id);
    }
});

// Methods
function getNamespaceDetailsFromNamespaceId(namespaceId) {
    console.log(namespaceId);
    
    getNamespaceDetails(namespaceId)
        .then((response) => {
            namespaceName.value = response.namespace?.name || '';
            usersNamespace.value = response.namespace?.users || [];
        })
        .catch((error) => {
            console.error("Error fetching namespace details:", error);
        });
}
function showConfirmDelete(user) {
    confirmDialog.value.open({
        title: t('administration.namespaces.dialogs.deleteUserFromNamespace.title'),
        message: t('administration.namespaces.dialogs.deleteUserFromNamespace.confirmationText', { userEmail: user.email }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteUserFromNamespace(user);
        }
    })
}
function deleteUserFromNamespace(user) {
    showOverlay();

    deleteUserFromNamespaceByUserId(props.namespace.id, user.id)
        .then(() => {
            showNotification(t('administration.namespaces.notifications.deleteUserSuccess'), 'success');
            emit('userDeletedFromNamespace');
        })
        .catch((error) => {
            console.error("Error deleting user from namespace:", error);
        })
        .finally(() => {
            hideOverlay();
        });
}
function copyToClipboard(text) {
    navigator.clipboard.writeText(text)
        .then(() => {
            copiedItem.value = text;
            setTimeout(() => {
                copiedItem.value = null;
            }, 2000);
        })
}
</script>
