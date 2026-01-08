@APIMetric(
    id = "coverage.m4demo",
    name = "M4(denominator)",
    lowerIsBetter = "false",
    aggregate = "total",        
    precision = "0"
)
@Description("M4 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m4DemoMetric(Partition src, Partition target) {
    Atom atom = src.getAtom();
    if (atom){
        Object value = atom.getProperty("coverage.m4demo");
        def parsed = parseNumber2(value);
        //out.println("M4 demo of " + src.getName() + " = " + parsed);
        return parsed;
    }
    return 0;
}

def parseNumber2(obj) {
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
   "M4Demo": "M4 Demo Coverage",
   "M4 coverage indicator from LDI import": "Coverage M4 value (from LDI import)"
]
