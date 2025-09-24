<template>
  <AuthLayout>
    <v-row align-content="center" justify="center" mx-auto class="my-12">
      <v-col cols="12" class="text-center mb-4">
        <v-img src="@/assets/logo.png" alt="Logo" max-width="100" class="mx-auto" />
        <BaseTitle :level="2" :title="$t('register.title')" class="mt-2" />
      </v-col>
      <v-card width="500" class="pa-2" elevation="8">
        <v-card-text>
          <v-form v-model="isValid">
            <BaseTextfield :Textfield="textfields.firstname" />
            <BaseTextfield :Textfield="textfields.lastname" />
            <BaseTextfield :Textfield="textfields.email" />
            <BaseTextfield :Textfield="textfields.password" :iconAction="passwordEyeIcon" @clickIcon="showPassword = !showPassword" />
            <BaseTextfield :Textfield="textfields.confirmPassword" :iconAction="confirmPasswordEyeIcon" @clickIcon="showConfirmPassword = !showConfirmPassword" />
          </v-form>
          <BaseButton :text="$t('register.buttons.register')" class="w-100 mt-4" :disabled="!isValid" @click="register" />
        </v-card-text>
        <v-card-actions class="d-flex flex-column align-center mt-4">
          <div class="d-flex align-center justify-center ga-1">
              <span>{{ $t('register.alreadyHaveAccount') }}</span>
              <RouterLink to="/login" class="text-router-link">
                  {{ $t('register.buttons.login') }}
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
import { ref, reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
const { t } = useI18n();

const { emailRules, passwordRules } = FormValidationRules();
const isValid = ref(false);
const showPassword = ref(false);
const showConfirmPassword = ref(false);

//Computed
const passwordEyeIcon = computed(() => showPassword.value ? 'mdi-eye' : 'mdi-eye-off');
const confirmPasswordEyeIcon = computed(() => showConfirmPassword.value ? 'mdi-eye' : 'mdi-eye-off');
const passwordType = computed(() => showPassword.value ? "text" : "password");
const confirmPasswordType = computed(() => showConfirmPassword.value ? "text" : "password");

// Reactive state
const textfields = reactive({
  firstname: new TextField({
    label: t('register.firstname'),
    type: "text",
    required: true
  }),
  lastname: new TextField({
    label: t('register.lastname'),
    type: "text",
    required: true
  }),
  email: new TextField({
    label: t('register.email'),
    type: "email",
    required: true,
    rules: emailRules
  }),
  password: new TextField({
    label: t('register.password'),
    type: passwordType,
    required: true,
    rules: passwordRules
  }),
  confirmPassword: new TextField({
    label: t('register.confirmPassword'),
    type: confirmPasswordType,
    required: true,
    rules: [(v) => v === textfields.password.value || t('rules.validation.password.mismatch')]
  })
});

// Methods
function register() {
  console.log("Form submitted:", textfields);
}
</script>

<style scoped>
.text-router-link {
    color: inherit;
    text-decoration: none;
}
</style>
