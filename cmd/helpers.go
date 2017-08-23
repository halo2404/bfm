package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func errorExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readFileContents(location string) (string, error) {
	bytes, err := ioutil.ReadFile(location)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func flagProvided(flags Flags) bool {
	return flags.Tap || flags.Brew || flags.Cask || flags.Mas
}

func getPackageType(flags Flags) string {
	if flags.Tap {
		return "tap"
	}

	if flags.Brew {
		return "brew"
	}

	if flags.Cask {
		return "cask"
	}

	if flags.Mas {
		return "mas"
	}

	return ""
}

func constructBaseEntry(packageType, packageName string) string {
	return fmt.Sprintf("%s '%s'", packageType, packageName)
}

func entryExists(contents, packageType, packageToCheck string) bool {
	packageEntry := constructBaseEntry(packageType, packageToCheck)

	if !strings.Contains(contents, packageEntry) {
		return false
	}

	return true
}

func getPackages(packageType string, lines []string) []string {
	var packages []string
	for _, line := range lines {
		if strings.HasPrefix(line, packageType) {
			packages = append(packages, line)
		}
	}

	return packages
}

//func constructFileContents(tap, brew, cask, mas []string) string {
//	lines := []string{}
//
//	for _, line := range tap {
//		lines = append(lines, line)
//	}
//
//	lines = append(lines, "")
//
//	for _, line := range brew {
//		lines = append(lines, line)
//	}
//
//	lines = append(lines, "")
//
//	for _, line := range cask {
//		lines = append(lines, line)
//	}
//
//	lines = append(lines, "")
//
//	for _, line := range mas {
//		lines = append(lines, line)
//	}
//
//	return strings.Join(lines, "\n")
//}
