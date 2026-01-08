@APIMetric(
    id = "coverage.m2_1_Percent",
    name = "M2_Result",
    lowerIsBetter = "false",       
    precision = "double",
    aggregate = "none"
)
@Description("M2_1 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m2_1PercentMetric(Partition src, Partition target) {
    def model = getModel();

    try {
        def m2Metric = model.getMetricDefinition("partition.metric.custom.coverage.m2");
        def m3DemoMetric = model.getMetricDefinition("partition.metric.custom.coverage.m3demo");

        if (!m2Metric || !m3DemoMetric) {
            out.println("M2 or M3Demo metric not found in the model.");
            return 0;
        }

        def m2MetricValue = model.getMetricValue(src, m2Metric);
        def m3DemoMetricValue = model.getMetricValue(src, m3DemoMetric);

        out.println("M2 Metric: " + m2MetricValue);
        out.println("M3 Demo Metric: " + m3DemoMetricValue);

        if (m3DemoMetricValue != 0){
            return (m3DemoMetricValue / m2MetricValue);
        }

    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M2_1 metric: " + e.getMessage());
    }
    return 0;
}

@Localize("en")
def en = [
   "M2_1Percent": "M2_1 Percent Coverage",
   "M2_1 coverage indicator from LDI import": "Coverage M2 value (from LDI import)"
]
