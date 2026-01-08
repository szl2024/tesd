@APIMetric(
    id = "coverage.m6_percent",
    name = "M6_Result",
    lowerIsBetter = "false",       
    precision = "percent",
    aggregate = "none"
)
@Description("M6 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m6PercentMetric(Partition src, Partition target) {
    def model = getModel();

    try {
        def m6Metric = model.getMetricDefinition("partition.metric.custom.coverage.m6");
        def m6DemoMetric = model.getMetricDefinition("partition.metric.custom.coverage.m6demo");

        if (!m6Metric || !m6DemoMetric) {
            out.println("M6 or M6 Demo metric not found in the model.");
            return 0;
        }

        def m6MetricValue = model.getMetricValue(src, m6Metric);
        def m6DemoMetricValue = model.getMetricValue(src, m6DemoMetric);

        out.println("M6 Metric: " + m6MetricValue);
        out.println("M6 Demo Metric: " + m6DemoMetricValue);

        if (m6DemoMetricValue != 0){
            return (m6MetricValue / m6DemoMetricValue);
        }
    
    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M6 metric: " + e.getMessage());
    }
    return 0;
}


@Localize("en")
def en = [
   "M6Percent": "M6 Percent Coverage",
   "M6 coverage indicator from LDI import": "Coverage M6 value (from LDI import)"
]
