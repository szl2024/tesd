@APIMetric(
    id = "coverage.m3",
    name = "M3(numerator)",
    lowerIsBetter = "false",
    aggregate = "total",        
    precision = "0"
)
@Description("M3 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m3Metric(Partition src, Partition target) {
    Atom atom = src.getAtom();
    if (atom){
        Object value = atom.getProperty("coverage.m3");

        def parsed = parseNumber(value);

        return parsed;

    }
    return 0;
}

def parseNumber(obj) {
    if (obj instanceof Number) {
        return obj;
    } else if (obj instanceof String) {
        try {
            return Double.parseDouble(obj);
        } catch (Exception e) {
            println("Invalid number format: " + obj);
        }
    }
    return 0;
}

@Localize("en")
def en = [
   "M3": "M3 Coverage",
   "M3 coverage indicator from LDI import": "Coverage M3 value (from LDI import)"
]
