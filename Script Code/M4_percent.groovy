@APIMetric(
    id = "coverage.m4_percent",
    name = "M4(Proportion of violations)",
    lowerIsBetter = "false",       
    precision = "percent",
    aggregate = "none"
)
@Description("M4 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m4PercentMetric(Partition src, Partition target) {
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

        if (m4DemoMetricValue != 0){
            return (m4MetricValue / m4DemoMetricValue);
        }
    
    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M4 metric: " + e.getMessage());
    }
    return 0;
}


@Localize("en")
def en = [
   "M4Percent": "M4 Percent Coverage",
   "M4 coverage indicator from LDI import": "Coverage M4 value (from LDI import)"
]
