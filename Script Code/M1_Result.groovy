@APIMetric(
    id = "coverage.m1",
    name = "M1_Result",
    lowerIsBetter = "false",
    aggregate = "none",        
    precision = "0"     
)
@Description("M1 coverage indicator from LDI import")
@Definition([
    "heatmap=true",
    "warning=1000",
    "error=1500"
])
@Group("coverage_metrics")
def m1Metric(Partition src, Partition target) {
    Atom atom = src.getAtom();
    if (atom){
        Object value = atom.getProperty("coverage.m1");
        println("value " + value);
        return value;
    }
    return 0;
}

@Localize("en")
def en = [
   "M1": "M1 Coverage",
   "M1 coverage indicator from LDI import": "Coverage M1 value (from LDI import)"
]


