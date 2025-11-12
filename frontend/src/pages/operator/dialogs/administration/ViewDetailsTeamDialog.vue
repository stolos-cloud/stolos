<template>
    <div class="view-details-team-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.teams.dialogs.viewDetailsTeam.title')" closable>
            <div class="d-flex align-center justify-center">
                <BaseTitle :level="3" :title="$t('administration.teams.dialogs.viewDetailsTeam.usersTitle')" />
                <v-spacer></v-spacer>
                <span>{{ $t('administration.teams.dialogs.viewDetailsTeam.totalMembersLabel', { count: usersTeam.length }) }}</span>
            </div>
            <v-virtual-scroll
                :items="usersTeam"
                max-height="200"
                v-if="usersTeam.length > 0"
            >
                <template v-slot:default="{ item }">
                    <v-list lines="two">
                        <v-list-item
                            :key="item.id"
                            :title="item.email"
                            class="border rounded"
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
                                <v-btn 
                                    v-tooltip="{ text: $t('administration.teams.buttons.deleteUserFromTeam') }" 
                                    icon="mdi-delete" 
                                    size="small" variant="text" 
                                    @click="showConfirmDelete(item)"
                                />
                            </template>
                        </v-list-item>
                    </v-list>
                </template>
            </v-virtual-scroll>
            <div v-else class="no-members">
                <p>{{ $t('administration.teams.dialogs.viewDetailsTeam.messages.noMembers') }}</p>
            </div>
        </BaseDialog>
        <BaseConfirmDialog ref="confirmDialog" />
    </div>
</template>

<script setup>
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { getTeamDetails, deleteUserFromTeamByUserId } from "@/services/teams.service";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    team: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);
const usersTeam = ref([]);
const confirmDialog = ref(null);
const copiedItem = ref(null);

// Emits
const emit = defineEmits(['update:modelValue', 'userDeletedFromTeam']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if(val && props.team) {
        getTeamDetailsFromTeamId(props.team.id);
    }
});

// Methods
function getTeamDetailsFromTeamId(teamId) {
    getTeamDetails(teamId)
        .then((teamDetails) => {
            const response = teamDetails;
            usersTeam.value = response.team?.users || [];            
        })
        .catch((error) => {
            console.error("Error fetching team details:", error);
        });
}
function showConfirmDelete(user) {
    confirmDialog.value.open({
        title: t('administration.teams.dialogs.deleteUserFromTeam.title'),
        message: t('administration.teams.dialogs.deleteUserFromTeam.confirmationText', { userEmail: user.email }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteUserFromTeam(user);
        }
    })
}
function deleteUserFromTeam(user) {
    showOverlay();

    deleteUserFromTeamByUserId(props.team.id, user.id)
        .then(() => {
            showNotification(t('administration.teams.notifications.deleteUserSuccess'), 'success');
            emit('userDeletedFromTeam');
        })
        .catch((error) => {
            console.error("Error deleting user from team:", error);
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