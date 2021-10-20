package cmd

func verbosePrintln(line string) {
	if verbose {
		println(line)
	}
}
