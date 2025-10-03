<template>
    <AuthLayout>
        <v-row align-content="center" justify="center" mx-auto class="my-12">
            <v-col cols="12" class="text-center mb-4">
                <v-img src="@/assets/logo.png" alt="Logo" max-width="100" class="mx-auto" />
                <BaseTitle :level="2" :title="$t('login.title')" class="mt-2" />
            </v-col>
            <v-card width="500" class="pa-2" elevation="8">
                <v-card-text>
                    <v-form v-model="isValid">
                        <BaseTextfield :Textfield="textfields.email" />
                        <BaseTextfield :Textfield="textfields.password" :iconAction="passwordEyeIcon" @clickIcon="showPassword = !showPassword" />
                    </v-form>
                    <BaseNotice v-if="sessionExpired" :text="$t('errors.sessionExpired')" type="error" closable />
                    <BaseNotice v-if="errorMessage" :text="errorMessage" type="error" />
                    <BaseButton :text="$t('login.buttons.login')" class="w-100 mt-4" :disabled="!isValid || isLoading" @click="loginUser" />
                </v-card-text>
                <v-card-actions class="d-flex flex-column align-center mt-4">
                    <RouterLink to="/403" class="mb-1 text-router-link">
                        {{ $t('login.forgotPassword') }}
                    </RouterLink>
                </v-card-actions>
            </v-card>
        </v-row>
    </AuthLayout>
</template>

<script setup>
import AuthLayout from "@/components/layouts/AuthLayout.vue";
import BaseTitle from "@/components/base/BaseTitle.vue";
import BaseTextfield from "@/components/base/BaseTextfield.vue";
import BaseButton from "@/components/base/BaseButton.vue";

import { TextField } from "@/models/TextField.js";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { ref, computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useStore } from "vuex";

const { t } = useI18n();
const { emailRules, passwordRules } = FormValidationRules();
const router = useRouter();
const store = useStore();

const props = defineProps({
  message: {
    type: String,
    default: ''
  }
});

// Validation state
const isValid = ref(false);
const isLoading = ref(false);
const errorMessage = ref('');
const showPassword = ref(false);

//Computed
const passwordEyeIcon = computed(() => showPassword.value ? 'mdi-eye' : 'mdi-eye-off');
const passwordType = computed(() => showPassword.value ? "text" : "password");
const sessionExpired = computed(() => props.message === 'sessionExpired');

// Form state
const textfields = reactive({
  email: new TextField({
    label: t('login.email'),
    type: "email",
    required: true,
    rules: emailRules
  }),
  password: new TextField({
    label: t('login.password'),
    type: passwordType,
    required: true,
    rules: passwordRules
  }),
});

// Methods
function loginUser() {
  if (!isValid.value) return;

  isLoading.value = true;
  errorMessage.value = '';
  
  store.dispatch('user/loginUser', {
    email: textfields.email.value,
    password: textfields.password.value
  })
  .then(() => {
    router.push('/dashboard');
  })
  .catch((error) => {
    handleLoginError(error);
  })
  .finally(() => {
    textfields.email.value = undefined;
    textfields.password.value = undefined;
    isLoading.value = false;
    errorMessage.value = '';
  });
}
//TODO : Put this methods into a error Composable in the futur if needed
function handleLoginError(error) {
  switch (error.message) {
    case 'failedLogin':
      errorMessage.value = t('errors.failedLogin');
      break;
    default:
      errorMessage.value = "An unexpected error occurred. Please try again later.";
  }
}
</script>

<style scoped>
.text-router-link {
    color: inherit;
    text-decoration: none;
}
</style>
