import { ref } from "vue";

const overlay = ref(false);

export function GlobalOverlayHandler() {
    function showOverlay() {
        overlay.value = true;
    }

    function hideOverlay() {
        overlay.value = false;
    }

    return {
        overlay,
        showOverlay,
        hideOverlay
    };
}