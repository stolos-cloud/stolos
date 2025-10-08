import { useI18n } from 'vue-i18n';

export function ErrorHandler() {
    const { t } = useI18n();

    function handleLoginError(error) {
        switch (error.message) {
            case 'failedLogin':
                return t('errors.failedLogin');
            default:
                return t('errors.unexpected');
        }
    }
    return {
        handleLoginError
    };
}