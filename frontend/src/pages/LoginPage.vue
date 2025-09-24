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
                    <BaseCheckbox :Checkbox="checkboxes.rememberMe" />
                    <BaseButton :text="$t('login.buttons.login')" class="w-100 mt-4" :disabled="!isValid" @click="login" />
                </v-card-text>
                <v-card-actions class="d-flex flex-column align-center mt-4">
                    <RouterLink to="/403" class="mb-1 text-router-link">
                        {{ $t('login.forgotPassword') }}
                    </RouterLink>
                    <div class="d-flex align-center justify-center ga-1">
                        <span>{{ $t('login.noAccount') }}</span>
                        <RouterLink to="/register" class="text-router-link">
                            {{ $t('login.buttons.signup') }}
                        </RouterLink>
                    </div>
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
import BaseCheckbox from "@/components/base/BaseCheckbox.vue";
import { TextField } from "@/models/TextField.js";
import { Checkbox } from "@/models/Checkbox.js";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { ref, computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

const { t } = useI18n();
const { emailRules, passwordRules } = FormValidationRules();
const router = useRouter();
const showPassword = ref(false);

//Computed
const passwordEyeIcon = computed(() => showPassword.value ? 'mdi-eye' : 'mdi-eye-off');
const passwordType = computed(() => showPassword.value ? "text" : "password");


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
const checkboxes = reactive({
    rememberMe: new Checkbox({
        label: t('login.rememberMe')
    })
});

// Validation state
const isValid = ref(false);

// Methods
function login() {
  router.push('/dashboard');
}
</script>

<style scoped>
.text-router-link {
    color: inherit;
    text-decoration: none;
}
</style>
