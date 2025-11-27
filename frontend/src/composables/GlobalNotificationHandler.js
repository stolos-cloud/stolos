import { ref } from "vue";

const notification = ref({
    visible: false,
    text: "",
    type: "info",
    closable: false,
});

export function GlobalNotificationHandler() {
    function showNotification(message, type, closable = false) {
        notification.value = {
            visible: true,
            text: message,
            type: type,
            closable: closable
        };
    }

    return {
        notification,
        showNotification,
    };
}