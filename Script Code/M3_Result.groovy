@APIMetric(
    id = "coverage.m3_a",
    name = "M3_Result",
    lowerIsBetter = "false",
    precision = "float",
    aggregate = "none"
)
@Description("M3_A indicator: M3 divided by log10(M3Demo + 1)")
@Definition([
    "heatmap=true",
    "warning=3.0",
    "error=5.0"
])
@Group("coverage_metrics")
def m3AMetric(Partition src, Partition target) {
    def model = getModel();

    try {
        def m3Metric = model.getMetricDefinition("partition.metric.custom.coverage.m3");
        def m3DemoMetric = model.getMetricDefinition("partition.metric.custom.coverage.m3demo");

        if (!m3Metric || !m3DemoMetric) {
            out.println("M3 or M3 Demo metric not found in the model.");
            return 0;
        }

        def m3MetricValue = model.getMetricValue(src, m3Metric);
        def m3DemoMetricValue = model.getMetricValue(src, m3DemoMetric);

        out.println("M3 Metric: " + m3MetricValue);
        out.println("M3 Demo Metric: " + m3DemoMetricValue);

        if (m3DemoMetricValue >= 0) {
            return m3MetricValue / Math.log10(m3DemoMetricValue + 1);
        }

    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M3_A metric: " + e.getMessage());
    }
    return 0;
}
