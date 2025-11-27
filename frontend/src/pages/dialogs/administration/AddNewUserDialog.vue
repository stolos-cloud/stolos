<template>
    <div class="add-new-user-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.users.dialogs.addUser.title')" closable>
            <v-form v-model="isValidForm">
                <BaseTextfield :Textfield="formFields.email" />
                <BaseTextfield :Textfield="formFields.password" />
                <BaseSelect v-model="formFields.role.value" :Select="formFields.role" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton :text="$t('actionButtons.add')" :disabled="!isValidForm" @click="addUser" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { Select } from "@/models/Select.js";
import { ref, reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { createNewUser } from '@/services/users.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const store = useStore();
const { emailRules, textfieldRules, passwordRules } = FormValidationRules();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    }
});

// State
const isValidForm = ref(false);
const isOpen = ref(props.modelValue);

// Computed
const userRoles = computed(() => store.getters['referenceLists/getUserRoles']);

// Form state
const formFields = reactive({
    email: new TextField({
        label: computed(() => t('administration.users.formfields.email')),
        type: "email",
        required: true,
        rules: emailRules
    }),
    password: new TextField({
        label: computed(() => t('administration.users.formfields.password')),
        type: "password",
        required: true,
        rules: passwordRules
    }),
    role: new Select({
        label: computed(() => t('administration.users.formfields.role')),
        options: userRoles.value,
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'newUserAdded']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.email.value = undefined;
    formFields.role.value = undefined;
    formFields.password.value = undefined;
    isOpen.value = false;
}
function addUser() {
    if (!isValidForm.value) return;
    showOverlay();

    const userData = {
        email: formFields.email.value,
        role: formFields.role.value,
        password: formFields.password.value
    };

    createNewUser(userData)
        .then((response) => {
            if(response?.user){
                showNotification(t('administration.users.notifications.addSuccess', { email: formFields.email.value }), 'success');
                emit('newUserAdded');
            }
        })
        .catch((error) => {
            console.error("Error adding user:", error);
            showNotification(t('administration.users.notifications.addError'), 'error');
        })
        .finally(() => {
            closeDialog();
            hideOverlay();
        });
} 
</script>