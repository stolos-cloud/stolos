<template>
    <AuthLayout>
        <v-row align-content="center" justify="center" mx-auto class="my-12">
            <v-col cols="12" class="text-center mb-4">
                <v-img src="@/assets/logo.png" alt="Logo" max-width="100" class="mx-auto" />
                <BaseTitle level="2" :title="$t('connexion.title')" class="mt-2" />
            </v-col>
            <v-card width="500" class="pa-2" elevation="8">
                <v-card-text>
                    <v-form v-model="isValid">
                        <BaseTextfield :Textfield="textfields.email" />
                        <BaseTextfield :Textfield="textfields.password" />
                    </v-form>
                    <v-checkbox
                        v-model="rememberMe"
                        :label="$t('connexion.rememberMe')"
                        class="mb-4"
                    />
                    <BaseButton :text="$t('connexion.buttons.login')" class="w-100" :disabled="!isValid" @click="handleSubmit" />
                </v-card-text>
                <v-card-actions class="d-flex flex-column align-center mt-4">
                    <RouterLink to="/403" class="mb-1 text-router-link">
                        {{ $t('connexion.forgotPassword') }}
                    </RouterLink>
                    <div class="d-flex align-center justify-center ga-1">
                        <span>{{ $t('connexion.noAccount') }}</span>
                        <RouterLink to="/creation-compte" class="text-router-link">
                            {{ $t('connexion.buttons.signup') }}
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
import { TextField } from "@/models/TextField.js";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { reactive } from "vue";
import { ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const { emailRules, passwordRules } = FormValidationRules();
// Form state
const textfields = reactive({
  email: new TextField({
    label: t('connexion.email'),
    type: "email",
    required: true,
    rules: emailRules
  }),
  password: new TextField({
    label: t('connexion.password'),
    type: "password",
    required: true,
    rules: passwordRules
  }),
});

// 
const isValid = ref(false);
const rememberMe = ref(false);

function handleSubmit() {
  console.log("Connexion form submitted:", {
    email: textfields.email.value,
    password: textfields.password.value,
    rememberMe: rememberMe.value,
  });
}
</script>

<style scoped>
.text-router-link {
    color: inherit;
    text-decoration: none;
}
</style>
