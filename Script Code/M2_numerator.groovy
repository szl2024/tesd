@APIMetric(
    id = "coverage.m2demo",
    name = "M2(numerator)",
    lowerIsBetter = "false",
    aggregate = "total",        
    precision = "0"
)
@Description("M2 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m3DemoMetric(Partition src, Partition target) {
    Atom atom = src.getAtom();
    if (atom){
        Object value = atom.getProperty("coverage.m3demo");
        def parsed = parseNumber2(value);
        //out.println("M3 demo of " + src.getName() + " = " + parsed);
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
   "M3Demo": "M3 Demo Coverage",
   "M3 coverage indicator from LDI import": "Coverage M3 value (from LDI import)"
]
