// Copyright © 2017 Jade Iqbal <jadeiqbal@fastmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"sort"

	"regexp"

	"io/ioutil"

	"errors"

	"github.com/lgug2z/bfm/brew"
	"github.com/lgug2z/bfm/brewfile"
	"github.com/spf13/cobra"
)

var addFlags Flags

func init() {
	RootCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addFlags.DryRun, "dry-run", "d", false, "conduct a dry run without modifying the Brewfile")

	addCmd.Flags().BoolVarP(&addFlags.Tap, "tap", "t", false, "add a tap")
	addCmd.Flags().BoolVarP(&addFlags.Brew, "brew", "b", false, "add a brew package")
	addCmd.Flags().BoolVarP(&addFlags.Cask, "cask", "c", false, "add a cask")
	addCmd.Flags().BoolVarP(&addFlags.Mas, "mas", "m", false, "add a mas app")

	addCmd.Flags().StringSliceVar(&addFlags.Args, "args", []string{}, "supply args to be used during installations and updates")
	addCmd.Flags().StringVar(&addFlags.RestartService, "restart-service", "", "always (every time bundle runs), changed (after changes and updates)")
	addCmd.Flags().StringVarP(&addFlags.MasID, "mas-id", "i", "", "id required for mas packages")

	addCmd.Flags().BoolVarP(&addFlags.AddPackageAndRequired, "required", "r", false, "add package and all required dependencies")
	addCmd.Flags().BoolVarP(&addFlags.AddAll, "all", "a", false, "add package and all required, recommended, optional and build dependencies")
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a dependency to your Brewfile",
	Long: `
Adds the dependency given as an argument to the Brewfile.

This command will modify your Brewfile without creating
a backup. Consider running the command with the --dry-run
flag if using bfm for the first time.

The type must be specified using the appropriate flag.

Taps must conform to the format <user/repo>.

Brew packages can have arguments specified using the --arg
flag (multiple arguments can be separated by using a comma),
and can specify service restart behaviour (always: restart
every time bundle is run, changed: only when updated or
changed) with the --restart-service flag.

MAS apps must specify an id using the --mas-id flag which
can be found by running 'mas search <app>'.

Examples:

bfm add -t homebrew/dupes
bfm add -b vim -a HEAD,with-override-system-vi
bfm add -b crisidev/chunkwm/chunkwm -r changed
bfm add -c macvim
bfm add -m Xcode -i 497799835

`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var cache brew.InfoCache
		var packages brewfile.Packages
		error := Add(args, &packages, cache, brewfilePath, brewInfoPath, addFlags)
		errorExit(error)
	},
}

func Add(args []string, packages *brewfile.Packages, cache brew.InfoCache, brewfilePath, brewInfoPath string, flags Flags) error {
	if !flagProvided(flags) {
		return errors.New("A package type must be specified. See 'bfm add -help'.")
	}

	toAdd := args[0]
	packageType := getPackageType(flags)

	error := packages.FromBrewfile(brewfilePath)
	errorExit(error)

	if entryExists(string(packages.Bytes()), packageType, toAdd) {
		return fmt.Errorf("%s '%s' is already in the Brewfile.", packageType, toAdd)
	}

	error = cache.Read(brewInfoPath)
	errorExit(error)

	cacheMap := brew.CacheMap{Cache: &cache, Map: make(brew.Map)}
	cacheMap.FromPackages(packages.Brew)
	cacheMap.ResolveRequiredDependencyMap()

	if flags.Tap {
		if !hasCorrectTapFormat(toAdd) {
			return fmt.Errorf("Unrecognised tap format. Use the format 'user/repo'.")
		}
		packages.Tap = addPackage(packageType, toAdd, packages.Tap, flags)
		sort.Strings(packages.Tap)
	}

	if flags.Brew {
		updated, error := addBrewPackage(toAdd, flags.RestartService, flags.Args, cacheMap, flags)
		if error != nil {
			return error
		}
		packages.Brew = updated
	}

	if flags.Cask {
		packages.Cask = addPackage(packageType, toAdd, packages.Cask, flags)
		sort.Strings(packages.Cask)
	}

	if flags.Mas {
		if !hasMasID(flags.MasID) {
			return fmt.Errorf("An id is required for mas apps. Get the id with 'mas search %s' and try again.", toAdd)
		}

		packages.Mas = addPackage(packageType, toAdd, packages.Mas, flags)
		sort.Strings(packages.Mas)
	}

	if flags.DryRun {
		fmt.Println(string(packages.Bytes()))
	} else {
		error := ioutil.WriteFile(brewfilePath, packages.Bytes(), 0644)
		errorExit(error)
	}

	return nil
}

func addBrewPackage(add, restart string, args []string, cacheMap brew.CacheMap, flags Flags) ([]string, error) {
	if len(restart) > 1 {
		switch restart {
		case "always":
			restart = "true"
		case "changed":
			restart = ":changed"
		default:
			return []string{}, errors.New("Valid options for the --restart-service flag are 'true' and 'changed'.")
		}
	}

	if flags.AddAll {
		if err := cacheMap.Add(brew.Entry{Name: add, RestartService: restart, Args: args}, brew.AddAll); err != nil {
			return []string{}, err
		}
	} else if flags.AddPackageAndRequired {
		if err := cacheMap.Add(brew.Entry{Name: add, RestartService: restart, Args: args}, brew.AddPackageAndRequired); err != nil {
			return []string{}, err
		}
	} else {
		if err := cacheMap.Add(brew.Entry{Name: add, RestartService: restart, Args: args}, brew.AddPackageOnly); err != nil {
			return []string{}, err
		}
	}

	lines := []string{}

	for _, b := range cacheMap.Map {
		entry, err := b.Format()
		if err != nil {
			return []string{}, err
		}

		lines = append(lines, entry)
	}

	sort.Strings(lines)
	return lines, nil
}

func addPackage(packageType, newPackage string, packages []string, flags Flags) []string {
	packageEntry := constructBaseEntry(packageType, newPackage)

	if packageType == "mas" {
		packageEntry = appendMasID(packageEntry, flags.MasID)
	}

	fmt.Printf("Added %s %s to Brewfile.\n", packageType, newPackage)
	return append(packages, packageEntry)
}

func hasCorrectTapFormat(tap string) bool {
	result, _ := regexp.MatchString(`.+/.+`, tap)
	return result
}

func hasMasID(i string) bool {
	return len(i) > 0
}

func appendMasID(packageEntry, i string) string {
	return fmt.Sprintf("%s, id: %s", packageEntry, i)
}
