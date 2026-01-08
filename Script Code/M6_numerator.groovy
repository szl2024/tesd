@APIMetric(id = "coverage.m6", name = "M6(numerator)", lowerIsBetter="false", aggregate="total", precision="0")
@Description("Aggregated M6 Coverage from atoms")
@Definition(["heatmap=true"])
@HelpURL("https://docs.lattix.com/lattix/userGuide/metrics.html")
def m6metric(Partition src, Partition target){
    def model = getModel();
    def atoms = model.getAtomsAt(src);

    double total = 0;
    if (atoms != null){
        for (atom in atoms){
            def value = atom.getProperty("coverage.m6");
            if (value instanceof Number){
                total += value;
            } else if (value instanceof String){
                try {
                    total += Double.parseDouble(value);
                } catch(Exception e){
                    println("Invalid number format in " + atom.getName() + ": " + value);
                }
            }
        }
    }
    return total;
}
