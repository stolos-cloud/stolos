<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('administration.users.title')"
            :subheading="$t('administration.users.subheading')"
        />
        <v-sheet border rounded class="mt-4">
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
              <v-toolbar flat>
                <v-toolbar-title>
                  {{ $t('administration.users.table.title') }}
                </v-toolbar-title>
                <BaseButton 
                  icon="mdi-plus" 
                  elevation="2" 
                  :tooltip="$t('administration.users.buttons.addUser')" 
                  :text="$t('administration.users.buttons.addUser')" 
                  @click="showAddUserDialog" 
                />
              </v-toolbar>
              <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined" hide-details single-line dense class="pa-3"/>
            </template>
            <template #item.role="{ item }">
                <div class="d-flex align-center">
                    <span>{{ item.role }}</span>
                    <v-icon
                        class="ml-2"
                        size="small"
                        icon="mdi-pencil"
                        @click="showEditRoleDialog(item)"
                    />
                </div>
            </template>
            <template #item.actions="{ item }">
              <v-btn v-tooltip="{ text: $t('administration.users.buttons.deleteUser') }" icon="mdi-delete" size="small" variant="text" @click="deleteUser(item)" />
            </template>
          </v-data-table>
        </v-sheet>
        <BaseDialog v-model="dialogEditUserRole" :title="$t('administration.users.dialogs.editUser.title')" width="600" closable>
          <v-form v-model="isValidEditUserForm">
            <BaseSelect v-model="formFields.role.value" :Select="formFields.role" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('actionButtons.cancel')" @click="closeEditRoleDialog" />
            <BaseButton size="small" :text="$t('actionButtons.edit')" :disabled="!isValidEditUserForm" @click="updateRole" />
          </template>
        </BaseDialog>
        <BaseDialog v-model="dialogAddUser" :title="$t('administration.users.dialogs.addUser.title')" width="600" closable>
          <v-form v-model="isValidAddUserForm">
            <BaseTextfield :Textfield="formFields.email" />
            <BaseTextfield :Textfield="formFields.password" />
            <BaseSelect v-model="formFields.role.value" :Select="formFields.role" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('actionButtons.cancel')" @click="closeAddUserDialog" />
            <BaseButton size="small" :text="$t('actionButtons.add')" :disabled="!isValidAddUserForm" @click="addUser" />
          </template>
        </BaseDialog>
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular
              indeterminate
            />
        </v-overlay>
        <BaseConfirmDialog ref="confirmDialog" />
        <BaseNotification v-model="notification.visible" :text="notification.text" :type="notification.type" />
    </PortalLayout>
</template>

<script setup>
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { Select } from "@/models/Select.js";
import { ref, reactive, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { getUsers, createNewUser, updateUserRole, deleteUserById } from '@/services/users.service';

const { t } = useI18n();
const store = useStore();
const { emailRules, textfieldRules, passwordRules } = FormValidationRules();

// State
const dialogEditUserRole = ref(false);
const dialogAddUser = ref(false);
const isValidEditUserForm = ref(false);
const isValidAddUserForm = ref(false);
const loading = ref(false);
const overlay = ref(false);
const users = ref([]);
const search = ref('');
const confirmDialog = ref(null);
const userTemp = ref(null);
const notification = ref({
  visible: false,
  text: '',
  type: 'info'
});

// Computed
const userHeaders = computed(() => [
  { title: t('administration.users.table.headers.email'), value: 'email'},
  { title: t('administration.users.table.headers.role'), value: 'role' },
  { title: t('administration.users.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);
const userRoles = computed(() => store.getters['referenceLists/getUserRoles']);

// Form state
const formFields = reactive({
  email: new TextField({
    label: t('administration.users.formfields.email'),
    type: "email",
    required: true,
    rules: emailRules
  }),
  password: new TextField({
    label: t('administration.users.formfields.password'),
    type: "password",
    required: true,
    rules: passwordRules
  }),
  role: new Select({
    label: t('administration.users.formfields.role'),
    options: userRoles.value,
    required: true,
    rules: textfieldRules
  })
});

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
  formFields.role.value = item.role;
  userTemp.value = item;
  dialogEditUserRole.value = true;
}

function showAddUserDialog() {
  formFields.email.value = undefined;
  formFields.role.value = undefined;
  formFields.password.value = undefined;
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

function addUser() {
    if (!isValidAddUserForm.value) return;

    overlay.value = true;
    const userData = {
        email: formFields.email.value,
        role: formFields.role.value,
        password: formFields.password.value
    };

    createNewUser(userData)
    .then(() => {
        showNotification(t('administration.users.notifications.addSuccess', { email: formFields.email.value }), 'success');
        fetchUsers();
    })
    .catch((error) => {
        console.error("Error adding user:", error);
        showNotification(t('administration.users.notifications.addError'), 'error');
    })
    .finally(() => {
        closeAddUserDialog();
        overlay.value = false;
    });
}

function updateRole() {
    if (!isValidEditUserForm.value) return;

    overlay.value = true;
    const userToUpdate = userTemp.value;
    
    updateUserRole(userToUpdate.id, formFields.role.value)
    .then(() => {
        showNotification(t('administration.users.notifications.updateSuccess', { email: userToUpdate.email }), 'success');
        fetchUsers();
    })
    .catch((error) => {
        console.error("Error updating user role:", error);
        showNotification(t('administration.users.notifications.updateError'), 'error');
    })
    .finally(() => {
        closeEditRoleDialog();
        overlay.value = false;
        userTemp.value = null;
    });
}   

function deleteUserConfirmed(item) {
  overlay.value = true;

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
    overlay.value = false;
  });
}

function closeEditRoleDialog() {
    formFields.role.value = undefined;
    dialogEditUserRole.value = false;
}

function closeAddUserDialog() {
    formFields.email.value = undefined;
    formFields.role.value = undefined;
    formFields.password.value = undefined;
    dialogAddUser.value = false;
}

function showNotification(message, type) {
  notification.value = {
    visible: true,
    text: message,
    type: type
  };
}
</script>