import { useI18n } from "vue-i18n";

export function FormValidationRules() {
    const { t } = useI18n();

    const emailRules = [
        (v) => !!v || t('rules.validation.email.required'),
        (v) => /.+@.+\..+/.test(v) || t('rules.validation.email.invalid'),
    ];
    const passwordRules = [
        (v) => !!v || t('rules.validation.password.required'),
        (v) => v.length >= 8 || t('rules.validation.password.invalid'),
        (v) => /[A-Z]/.test(v) || t('rules.validation.password.uppercase'),
        (v) => /[a-z]/.test(v) || t('rules.validation.password.lowercase'),
        (v) => /[0-9]/.test(v) || t('rules.validation.password.number'),
        (v) => /[!@#$%^&*(),.?":{}|<>-]/.test(v) || t('rules.validation.password.special')
    ];
    const textfieldRules = [
        (v) => !!v || t('rules.validation.textfield.required'),
    ];
    return {
        emailRules,
        passwordRules,
        textfieldRules
    };
}