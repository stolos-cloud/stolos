<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('administration.users.title')"
            :subheading="$t('administration.users.subheading')"
        />
        <v-sheet class="mt-4 border rounded">
            <v-data-table
                :headers="userHeaders"
                :items="users"
                :items-length="users.length"
                :loading=loading
                :search="search"
                :loading-text="$t('administration.users.table.loadingText')"
                :no-data-text="$t('administration.users.table.noDataText')"
                :items-per-page="10"
                :items-per-page-text="$t('administration.users.table.itemsPerPageText')"
                :hide-default-footer="users.length < 10"
                mobile-breakpoint="md"
            >
                <template v-slot:top>
                    <BaseToolbarTable v-model="search" :title="$t('administration.users.table.title')" :buttons="actionsButtonForTable" />
                </template>
                <template #[`item.role`]="{ item }">
                    <div class="d-flex align-center">
                        <span>{{ item.role }}</span>
                        <v-icon
                            v-if="currentUserId !== item.id"
                            class="ml-2"
                            size="small"
                            icon="mdi-pencil"
                            @click="showEditRoleDialog(item)"
                        />
                    </div>
                </template>
                <template #[`item.actions`]="{ item }">
                    <v-btn v-tooltip="{ text: $t('administration.users.buttons.deleteUser') }" icon="mdi-delete" size="small" variant="text" :disabled="currentUserId === item.id" @click="deleteUser(item)" />
                </template>
            </v-data-table>
        </v-sheet>
        <EditUserRoleDialog v-model="dialogEditUserRole" :userSelected="userTemp" @userRoleUpdated="fetchUsers" @update:userSelected="userTemp = $event" />
        <AddNewUserDialog v-model="dialogAddUser" @userAdded="fetchUsers" />
        <BaseConfirmDialog ref="confirmDialog" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { getUsers, deleteUserById } from '@/services/users.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import EditUserRoleDialog from "./dialogs/administration/EditUserRoleDialog.vue";
import AddNewUserDialog from "./dialogs/administration/AddNewUserDialog.vue";

const { t } = useI18n();
const store = useStore();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

// State
const dialogEditUserRole = ref(false);
const dialogAddUser = ref(false);
const loading = ref(false);
const users = ref([]);
const search = ref('');
const confirmDialog = ref(null);
const userTemp = ref(null);

// Computed
const userHeaders = computed(() => [
    { title: t('administration.users.table.headers.email'), value: 'email'},
    { title: t('administration.users.table.headers.role'), value: 'role' },
    { title: t('administration.users.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);
const currentUserId = computed(() => store.getters['user/getId']);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchUsers
    },
    {
        icon: "mdi-plus",
        tooltip: t('administration.users.buttons.addUser'),
        text: t('administration.users.buttons.addUser'),
        click: showAddUserDialog
    },
]);

//Mounted
onMounted(() => {
    fetchUsers();
});

//Methods
function fetchUsers() {
    getUsers().then(response => {
        users.value = response.users.map(user => ({
            ...user,
            role: user.role.charAt(0).toUpperCase() + user.role.slice(1)
        }));
    }).catch(error => {
        console.error("Error fetching users:", error);
    });
}
function showEditRoleDialog(item) {
    userTemp.value = item;
    dialogEditUserRole.value = true;
}
function showAddUserDialog() {
    dialogAddUser.value = true;
}
function deleteUser(item) {
    confirmDialog.value.open({
        title: t('administration.users.dialogs.deleteUser.title'),
        message: t('administration.users.dialogs.deleteUser.confirmationText', { email: item.email }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteUserConfirmed(item);
        }
    })
}
function deleteUserConfirmed(item) {
    showOverlay();

    deleteUserById(item.id)
        .then(() => {
            showNotification(t('administration.users.notifications.deleteSuccess', { email: item.email }), 'success');
            fetchUsers();
        })
        .catch((error) => {
            console.error("Error deleting user:", error);
            showNotification(t('administration.users.notifications.deleteError'), 'error');
        })
        .finally(() => {
            hideOverlay();
        });
}
</script>