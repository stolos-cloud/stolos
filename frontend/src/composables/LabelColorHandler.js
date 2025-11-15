/**
 * Composable for label color generation
 */
export const LabelColorHandler = () => {
    const colors = [
        'blue', 'green', 'orange', 'purple', 'cyan',
        'pink', 'teal', 'indigo', 'lime', 'amber',
        'deep-orange', 'light-blue', 'deep-purple', 'light-green'
    ];

    /**
     * Generate color for a label based on its text
     * @param {string} label - The label text
     * @returns {string} - Vuetify color name
     */
    const getLabelColor = (label) => {
        if (!label) return 'grey';

        // Hash function to get color for same label
        let hash = 0;
        for (let i = 0; i < label.length; i++) {
            hash = label.charCodeAt(i) + ((hash << 5) - hash);
        }

        const index = Math.abs(hash) % colors.length;
        return colors[index];
    };

    return {
        getLabelColor
    };
};
