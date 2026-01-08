@APIMetric(
    id = "coverage.m4_a",
    name = "M4_Result",
    lowerIsBetter = "false",
    precision = "float",
    aggregate = "none"
)
@Description("M4_A indicator: M4 divided by log10(M4Demo + 1)")
@Definition([
    "heatmap=true",
    "warning=3.0",
    "error=5.0"
])
@Group("coverage_metrics")
def m4AMetric(Partition src, Partition target) {
    def model = getModel();

    try {
        def m4Metric = model.getMetricDefinition("partition.metric.custom.coverage.m4");
        def m4DemoMetric = model.getMetricDefinition("partition.metric.custom.coverage.m4demo");

        if (!m4Metric || !m4DemoMetric) {
            out.println("M4 or M4 Demo metric not found in the model.");
            return 0;
        }

        def m4MetricValue = model.getMetricValue(src, m4Metric);
        def m4DemoMetricValue = model.getMetricValue(src, m4DemoMetric);

        out.println("M4 Metric: " + m4MetricValue);
        out.println("M4 Demo Metric: " + m4DemoMetricValue);

        if (m4DemoMetricValue >= 0) {
            return m4MetricValue / Math.log10(m4DemoMetricValue + 1);
        }

    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M4_A metric: " + e.getMessage());
    }
    return 0;
}
