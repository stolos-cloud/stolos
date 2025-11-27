/**
 * Composable for label color generation
 */
export const LabelColorHandler = () => {
    const colors = [
        'blue', 'green', 'orange', 'purple', 'cyan',
        'pink', 'teal', 'indigo', 'lime', 'amber',
        'deep-orange', 'light-blue', 'deep-purple', 'light-green'
    ];

    const getLabelColor = (label) => {
        if (!label) return 'grey';

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
