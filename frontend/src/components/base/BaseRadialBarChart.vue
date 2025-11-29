<template>
    <apexchart type="radialBar" height="250" :options="chartOptions" :series="percentageSeries" />
</template>

<script setup>
import { reactive, computed } from 'vue'
import VueApexCharts from 'vue3-apexcharts'

const apexchart = VueApexCharts

const props = defineProps({
    series: {
        type: Array,
        required: true
    },
    labels: {
        type: Array,
        required: true
    },
    colors: {
        type: Array,
        required: true
    }
})

// Computed
const percentageSeries = computed(() => {
    const total = props.series.reduce((sum, val) => sum + val, 0)
    return props.series.map(val => Math.round((val / total) * 100))
});

// Reactive
const chartOptions = reactive({
    plotOptions: {
        radialBar: {
            track: {
                background: 'rgba(var(--v-theme-on-surface), 0.1)',
                margin: 5,
            },
            hollow: {
                size: `${80 - props.series.length * 4}%`
            },
            dataLabels: {
                name: { show: true },
                value: {
                    show: true,
                    color: 'rgba(var(--v-theme-on-surface), 1)',
                },
                total: {
                    show: true,
                    showAlways: true,
                    label: 'Total',
                    color: 'rgba(var(--v-theme-on-surface), 1)',
                    formatter: function () {
                        return props.series.reduce((a, b) => a + b, 0)
                    }
                }
            },
        }
    },
    fill: {
        type: 'solid',
        colors: props.colors
    },
    labels: props.labels,
    legend: {
        show: true,
        position: 'right',
        labels: {
            useSeriesColors: false,
            colors: 'rgba(var(--v-theme-on-surface), 1)'
        },
        markers: {
            offsetX: -5,
            size: 6,
            shape: 'square',
            strokeWidth: 0
        },
        formatter: function (seriesName, opts) {
            return `${seriesName} <b>${props.series[opts.seriesIndex]}</b>`
        },
        itemMargin: {
            horizontal: 10,
            vertical: 5
        }
    },
    stroke: {
        lineCap: 'round',
    },
    states: {
        hover: {
            enabled: false
        },
        active: {
            allowMultipleDataPointsSelection: false,
            filter: {
                type: 'none'
            }
        }
    },
    responsive: [
        {
            breakpoint: 960,
            options: {
                legend: {
                    position: 'bottom'
                }
            }
        }
    ]
})
</script>
