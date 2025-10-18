<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('administration.teams.title')" :subheading="$t('administration.teams.subheading')" />
        <v-sheet border rounded class="mt-4">
            <v-data-table 
                :headers="teamHeaders" 
                :items="teams" 
                :items-length="teams.length" 
                :loading=loading
                :search="search" :loading-text="$t('administration.teams.table.loadingText')"
                :no-data-text="$t('administration.teams.table.noDataText')" :items-per-page="10"
                :items-per-page-text="$t('administration.teams.table.itemsPerPageText')" 
                mobile-breakpoint="md" :hide-default-footer="teams.length < 10"
                @click:row="(event, item) => {
                    if (event.target.closest('.v-btn, .v-icon')) return
                    showViewDetailsDialog(item.item)
                }"
            >
                <template v-slot:top>
                    <v-toolbar flat>
                        <v-toolbar-title>
                            {{ $t('administration.teams.table.title') }}
                        </v-toolbar-title>
                        <BaseButton 
                            icon="mdi-plus" 
                            elevation="2" 
                            :tooltip="$t('administration.teams.buttons.createNewTeam')"
                            :text="$t('administration.teams.buttons.createNewTeam')" 
                            @click="showCreateTeamDialog" />
                    </v-toolbar>
                    <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined"
                        hide-details single-line dense class="pa-3" />
                </template>
                <template #item.actions="{ item }">
                    <v-btn v-tooltip="{ text: $t('administration.teams.buttons.addUserToTeam') }" icon="mdi-account-plus" size="small" variant="text" @click="showAddUserToTeamDialog(item)" />
                    <v-btn v-tooltip="{ text: $t('administration.teams.buttons.deleteTeam') }" icon="mdi-delete" size="small" variant="text" @click="deleteTeam(item)" />
                </template>
            </v-data-table>
        </v-sheet>
        <CreateTeamDialog 
            v-model="dialogCreateTeam"
            @loading="overlay = $event"
            @teamCreated="teamCreated"
        />
        <AddUserToTeamDialog 
            v-model="dialogAddUserToTeam"
            :team="selectedTeam"
            @loading="overlay = $event"
            @userAdded="userAddedToTeam"
        />
        <ViewDetailsTeamDialog 
            v-model="dialogViewDetailsTeam"
            :team="selectedTeam"
            @loading="overlay = $event"
            @userDeletedFromTeam="userDeletedFromTeam"
        />
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular indeterminate />
        </v-overlay>
        <BaseConfirmDialog ref="confirmDialog" />
        <BaseNotification v-model="notification.visible" :text="notification.text" :type="notification.type" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTeams, deleteTeamById } from '@/services/teams.service';
import CreateTeamDialog from "./administration/dialogs/CreateTeamDialog.vue";
import AddUserToTeamDialog from "./administration/dialogs/AddUserToTeamDialog.vue";
import ViewDetailsTeamDialog from "./administration/dialogs/ViewDetailsTeamDialog.vue";

const { t } = useI18n();

// State
const dialogCreateTeam = ref(false);
const dialogAddUserToTeam = ref(false);
const dialogViewDetailsTeam = ref(false);
const loading = ref(false);
const overlay = ref(false);
const teams = ref([]);
const search = ref('');
const confirmDialog = ref(null);
const selectedTeam = ref(null);
const notification = ref({
    visible: false,
    text: '',
    type: 'info'
});

// Computed
const teamHeaders = computed(() => [
    { title: t('administration.teams.table.headers.teams'), value: 'name' },
    { title: t('administration.teams.table.headers.numberOfMembers'), value: 'numberOfUsers', sortable: false, align: 'center' },
    { title: t('administration.teams.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);

//Mounted
onMounted(() => {
    fetchTeams();
});

// Methods
function showCreateTeamDialog() {
    dialogCreateTeam.value = true;
}
function showAddUserToTeamDialog(item) {
    selectedTeam.value = item;
    dialogAddUserToTeam.value = true;
}
function showViewDetailsDialog(item) {
    selectedTeam.value = item;
    dialogViewDetailsTeam.value = true;
}
function teamCreated() {
    showNotification(t('administration.teams.notifications.createTeamSuccess'), 'success');
    fetchTeams();
}
function userAddedToTeam() {
    showNotification(t('administration.teams.notifications.addUserSuccess'), 'success');
    fetchTeams();
}
function userDeletedFromTeam() {
    showNotification(t('administration.teams.notifications.deleteUserSuccess'), 'success');
    fetchTeams();
}
function fetchTeams() {
    getTeams().then(response => {
        teams.value = response.teams
            .filter(team => team.name !== "administrators")
            .map(team => ({
                ...team,
                numberOfUsers: team.users?.length || 0
            }));
    }).catch(error => {
        console.error("Error fetching teams:", error);
    });
}
function deleteTeam(team) {
    confirmDialog.value.open({
        title: t('administration.teams.dialogs.deleteTeam.title'),
        message: t('administration.teams.dialogs.deleteTeam.confirmationText', { teamName: team.name }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteTeamConfirmed(team);
        }
    })
}
function deleteTeamConfirmed(team) {
    overlay.value = true;

    deleteTeamById(team.id)
    .then(() => {
        showNotification(t('administration.teams.notifications.deleteTeamSuccess'), 'success');
        fetchTeams();
    })
    .catch((error) => {
        console.error("Error deleting team:", error);
        showNotification(t('administration.teams.notifications.deleteTeamError'), 'error');
    })
    .finally(() => {
        overlay.value = false;
    });
}
function showNotification(message, type) {
    notification.value = {
        visible: true,
        text: message,
        type: type
    };
}
</script>

<style scoped>
:deep(.v-data-table__tr:hover) {
  background-color: rgba(var(--v-theme-on-surface), 0.1);
}
</style>