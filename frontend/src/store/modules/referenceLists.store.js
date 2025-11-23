export const referenceLists = {
    namespaced: true,
    state: {
        roleUserTypes: [
            { label: 'Admin', value: 'admin' },
            { label: 'Developer', value: 'developer' },
        ],
        roleProvisioningTypes: [
            { label: 'Worker', value: 'worker' },
            { label: 'Control plane', value: 'control-plane' },
        ],
        isoTypes: [
            { label: 'AMD64', value: 'amd64' },
            { label: 'ARM64', value: 'arm64' },
        ],
        diskTypes: [
            { label: 'Standard Persistent Disk (pd-standard)', value: 'pd-standard' },
            { label: 'Balanced Persistent Disk (pd-balanced)', value: 'pd-balanced' },
            { label: 'SSD Persistent Disk (pd-ssd)', value: 'pd-ssd' },
            { label: 'Extreme Persistent Disk (pd-extreme)', value: 'pd-extreme' },
        ],
        cloudZones: [],
        machinesTypesByZone: {},
        scaffolds: [],
    },
    mutations: {
        SET_CLOUD_ZONES(state, zones) {
            state.cloudZones = zones;
        },
        SET_MACHINE_TYPES_BY_ZONE(state, machineTypes) {
            state.machinesTypesByZone = machineTypes;
        },
        SET_SCAFFOLDS(state, scaffolds) {
            state.scaffolds = scaffolds;
        },
    },
    actions: {
        setCloudResources({ commit }, gcpResources) {
            commit(
                'SET_CLOUD_ZONES',
                gcpResources.zones.map(zone => ({ label: zone, value: zone }))
            );
            commit('SET_MACHINE_TYPES_BY_ZONE', gcpResources.machine_types_by_zone);
        },
        setScaffolds({ commit }, scaffolds) {
            commit(
                'SET_SCAFFOLDS',
                scaffolds.map(scaffold => ({
                    label: scaffold.charAt(0).toUpperCase() + scaffold.slice(1),
                    value: scaffold,
                }))
            );
        },
    },
    getters: {
        getUserRoles: state => state.roleUserTypes,
        getProvisioningRoles: state => state.roleProvisioningTypes,
        getIsoTypes: state => state.isoTypes,
        getDiskTypes: state => state.diskTypes,
        getCloudZones: state => state.cloudZones,
        getMachinesTypesByZone: state => zone => state.machinesTypesByZone[zone] || [],
        getScaffolds: state => state.scaffolds,
    },
};
