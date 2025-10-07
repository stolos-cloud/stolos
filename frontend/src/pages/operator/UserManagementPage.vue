<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('userManagement.title')"
            :subheading="$t('userManagement.subheading')"
        />
        <v-sheet border rounded class="mt-4">
          <v-data-table
            :headers="userHeaders"
            :items="users"
            :items-length="users.length"
            :loading=loading
            :search="search"
            :loading-text="$t('userManagement.table.loadingText')"
            :no-data-text="$t('userManagement.table.noDataText')"
            :items-per-page="10"
            :items-per-page-text="$t('userManagement.table.itemsPerPageText')"
            class="elevation-8"
            mobile-breakpoint="md"
            :hide-default-footer="users.length < 10"
          >
            <template v-slot:top>
              <v-toolbar flat>
                <v-toolbar-title>
                  {{ $t('userManagement.table.title') }}
                </v-toolbar-title>
                <BaseButton icon="mdi-plus" size="small" :text="$t('userManagement.buttons.addUser')" @click="showDialogAddUser" />
              </v-toolbar>
              <v-text-field
                v-model="search"
                label="Search"
                prepend-inner-icon="mdi-magnify"
                variant="outlined"
                hide-details
                single-line
                dense
                class="pa-3"
              />
            </template>
            <template v-slot:item.role="{ item }">
                <div class="d-flex align-center">
                    <span>{{ item.role }}</span>
                    <v-icon
                        class="ml-2"
                        size="small"
                        icon="mdi-pencil"
                        @click="editRole(item)"
                    />
                </div>
            </template>
            <template v-slot:item.actions="{ item }">
              <!-- <v-icon color="medium-emphasis" icon="mdi-delete" size="small" @click="deleteUser(item)"></v-icon> -->
            </template>
          </v-data-table>
        </v-sheet>
        <BaseDialog v-model="dialogEditUser" :title="$t('userManagement.dialogs.editUser.title')" width="600" closable>
          <v-form v-model="isValidEditUserForm">
            <BaseTextfield :Textfield="formFields.email" />
            <BaseTextfield :Textfield="formFields.role" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('userManagement.buttons.cancel')" @click="cancelEditRole" />
            <BaseButton size="small" :text="$t('userManagement.buttons.editUser')" :disabled="!isValidEditUserForm" @click="updateUserRole" />
          </template>
        </BaseDialog>
        <BaseDialog v-model="dialogAddUser" :title="$t('userManagement.dialogs.addUser.title')" width="600" closable>
          <v-form v-model="isValidAddUserForm">
            <BaseTextfield :Textfield="formFields.email" />
            <BaseTextfield :Textfield="formFields.role" />
            <BaseTextfield :Textfield="formFields.password" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('userManagement.buttons.cancel')" @click="cancelAddUser" />
            <BaseButton size="small" :text="$t('userManagement.buttons.addUser')" :disabled="!isValidAddUserForm" @click="addUser" />
          </template>
        </BaseDialog>
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular
              indeterminate
            />
        </v-overlay>
        <BaseNotification v-model="notification.visible" :text="notification.text" :type="notification.type" />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { ref, reactive, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getUsers } from '@/services/users.service';

const { t } = useI18n();
const { emailRules, textfieldRules, passwordRules } = FormValidationRules();

// State
const dialogEditUser = ref(false);
const dialogAddUser = ref(false);
const isValidEditUserForm = ref(false);
const isValidAddUserForm = ref(false);
const loading = ref(false);
const overlay = ref(false);
const users = ref([]);
const search = ref('');
const notification = ref({
  visible: false,
  text: '',
  type: 'info'
});

// Form state
const formFields = reactive({
  email: new TextField({
    label: t('userManagement.formfields.email'),
    type: "email",
    required: true,
    rules: emailRules
  }),
  password: new TextField({
    label: t('userManagement.formfields.password'),
    type: "password",
    required: true,
    rules: passwordRules
  }),
  role: new TextField({
    label: t('userManagement.formfields.role'),
    type: "text",
    required: true,
    rules: textfieldRules
  })
});

//Mounted
onMounted(() => {
  fetchUsers();
});

// Computed
const userHeaders = computed(() => [
  { title: t('userManagement.table.headers.email'), value: 'email'},
  { title: t('userManagement.table.headers.role'), value: 'role' },
  { title: t('userManagement.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);

//Methods
function fetchUsers() {
  getUsers().then(response => {
    users.value = response.users;
  }).catch(error => {
    console.error("Error fetching users:", error);
  });
}

function editRole(item) {
  formFields.role.value = item.role;
  dialogEditUser.value = true;
}

function cancelEditRole() {
    formFields.role.value = undefined;
    dialogEditUser.value = false;
}

function showDialogAddUser() {
  formFields.email.value = undefined;
  formFields.role.value = undefined;
  formFields.password.value = undefined;
  dialogAddUser.value = true;
}

function cancelAddUser() {
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