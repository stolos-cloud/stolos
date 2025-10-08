export const referenceLists = {
    namespaced: true,
    state: {
        roleUserTypes: [
            { label: 'Admin', value: 'admin' },
            { label: 'Developer', value: 'developer' }
        ],
        roleProvisioningTypes: [
            { label: 'Worker', value: 'worker' },
            { label: 'Control plane', value: 'control-plane' }
        ],
        isoTypes: [
            { label: 'AMD', value: 'amd' },
            { label: 'ARM', value: 'arm' },
        ],
        cloudZones: [],
        machinesTypesByZone: {}
    },
    mutations: {
        SET_CLOUD_ZONES(state, zones) {
            state.cloudZones = zones;
        },
        SET_MACHINE_TYPES_BY_ZONE(state, machineTypes) {
            state.machinesTypesByZone = machineTypes;
        }
    },
    actions: {
        setCloudResources({ commit }, gcpResources) {
            commit('SET_CLOUD_ZONES', gcpResources.zones);
            commit('SET_MACHINE_TYPES_BY_ZONE', gcpResources.machine_types_by_zone);
        },
    },
    getters: {
        getUserRoles: (state) => state.roleUserTypes,
        getProvisioningRoles: (state) => state.roleProvisioningTypes,
        getIsoTypes: (state) => state.isoTypes,
        getCloudZones: (state) => state.cloudZones,
        getMachinesTypesByZone: (state) => (zone) => state.machinesTypesByZone[zone] || []

    }
};