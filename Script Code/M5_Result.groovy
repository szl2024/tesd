@APIMetric(
    id = "coverage.m5_percent",
    name = "M5_Result",
    lowerIsBetter = "false",       
    precision = "percent",
    aggregate = "none"
)
@Description("M5 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m5PercentMetric(Partition src, Partition target) {
    def model = getModel();

    try {
        def m5Metric = model.getMetricDefinition("partition.metric.custom.coverage.m5");
        def m5DemoMetric = model.getMetricDefinition("partition.metric.custom.coverage.m5demo");

        if (!m5Metric || !m5DemoMetric) {
            out.println("M5 or M5 Demo metric not found in the model.");
            return 0;
        }

        def m5MetricValue = model.getMetricValue(src, m5Metric);
        def m5DemoMetricValue = model.getMetricValue(src, m5DemoMetric);

        out.println("M5 Metric: " + m5MetricValue);
        out.println("M5 Demo Metric: " + m5DemoMetricValue);

        if (m5DemoMetricValue != 0){
            return (m5MetricValue / m5DemoMetricValue);
        }
    
    } catch (Exception e) {
        e.printStackTrace();
        out.println("Error calculating M5 metric: " + e.getMessage());
    }
    return 0;
}


@Localize("en")
def en = [
   "M5Percent": "M5 Percent Coverage",
   "M5 coverage indicator from LDI import": "Coverage M5 value (from LDI import)"
]
