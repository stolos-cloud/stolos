export const StorageService = {
    set(key, value) {
        try {
            if (typeof value === 'string') {
                localStorage.setItem(key, value);
            } else {
                localStorage.setItem(key, JSON.stringify(value));
            }
        } catch (err) {
            console.error('Storage error:', err);
        }
    },
    get(key) {
        try {
            const value = localStorage.getItem(key);
            if (!value) return null;
            if (typeof value === 'string') {
                return value;
            }
            return JSON.parse(value);
        } catch (err) {
            console.error('Storage error:', err);
            return null;
        }
    },
    remove(key) {
        localStorage.removeItem(key);
    },
};
