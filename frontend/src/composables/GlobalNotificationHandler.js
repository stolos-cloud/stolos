import { ref } from "vue";

const notification = ref({
    visible: false,
    text: "",
    type: "info"
});

export function GlobalNotificationHandler() {
    function showNotification(message, type) {
        notification.value = {
            visible: true,
            text: message,
            type: type
        };
    }

    return {
        notification,
        showNotification,
    };
}