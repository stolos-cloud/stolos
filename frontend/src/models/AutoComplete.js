export class AutoComplete {
    constructor({ label, value, items, multiple, required, rules, noDataText }) {
        this.label = label;
        this.value = value;
        this.items = items;
        this.multiple = multiple;
        this.required = required;
        this.rules = rules;
        this.noDataText = noDataText;
    }
    getLabel() {
        return this.label;
    }

    getValue() {
        return this.value;
    }

    getItems() {
        return this.items;
    }

    isRequired() {
        return this.required;
    }

    isMultiple() {
        return this.multiple;
    }

    getRules() {
        return this.rules;
    }

    getNoDataText() {
        return this.noDataText;
    }

    setLabel(label) {
        this.label = label;
    }

    setValue(value) {
        this.value = value;
    }

    setItems(items) {
        this.items = items;
    }

    setRequired(required) {
        this.required = required;
    }

    setMultiple(multiple) {
        this.multiple = multiple;
    }

    setRules(rules) {
        this.rules = rules;
    }

    setNoDataText(noDataText) {
        this.noDataText = noDataText;
    }

    change(value) {
        this.value = value;
    }
}
