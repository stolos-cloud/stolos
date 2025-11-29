export function StatusColorHandler() {
    function getStatusColor(status) {
        switch (status.toLowerCase()) {
            case 'active':
            case 'healthy':
                return 'rgba(var(--v-theme-success), 1)';
            case 'provisioning':
                return 'rgba(var(--v-theme-info), 1)';
            case 'pending':
                return 'rgba(var(--v-theme-warning), 1)';
            case 'failed':
                return 'rgba(var(--v-theme-error), 1)';
            default:
                return 'rgba(var(--v-theme-primary), 1)';
        }
    }

    return {
        getStatusColor,
    };
}