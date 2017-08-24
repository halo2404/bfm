package cmd_test

import (
	. "github.com/lgug2z/bfm/cmd"

	"fmt"
	"os"

	"github.com/lgug2z/bfm/brew"
	"github.com/lgug2z/bfm/brewfile"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
)

var _ = Describe("Clean", func() {
	var (
		bf       = fmt.Sprintf("%s/%s", os.Getenv("GOPATH"), "src/github.com/lgug2z/bfm/testData/testBrewfile")
		dbFile   = fmt.Sprintf("%s/%s", os.Getenv("GOPATH"), "src/github.com/lgug2z/bfm/testData/testDB.bolt")
		cache    brew.InfoCache
		packages brewfile.Packages
		contents = `
tap 'homebrew/bundle'
brew 'a2ps'
tap 'homebrew/core'
cask 'google-chrome'
mas 'Xcode', id: 497799835
cask 'firefox'
# some comment
`
	)

	Describe("When the command is called", func() {
		It("Should read in the packages currently in the Brewfile", func() {
			testDB, err := NewTestDB(dbFile)
			Expect(err).ToNot(HaveOccurred())
			defer testDB.Close()

			testDB.AddTestBrewsByName("a2ps")

			t := TestFile{Path: bf, Contents: contents}
			Expect(t.Create()).To(Succeed())
			defer t.Remove()

			expectedPackages := brewfile.Packages{
				Tap:  []string{"tap 'homebrew/bundle'", "tap 'homebrew/core'"},
				Brew: []string{"brew 'a2ps'"},
				Cask: []string{"cask 'firefox'", "cask 'google-chrome'"},
				Mas:  []string{"mas 'Xcode', id: 497799835"},
			}

			Clean([]string{}, &packages, cache, bf, Flags{DryRun: false}, testDB.DB)

			Expect(packages).To(Equal(expectedPackages))
		})

		It("Should not proceed if a package in the Brewfile is not in the BoltDB cache", func() {
			testDB, err := NewTestDB(dbFile)
			Expect(err).ToNot(HaveOccurred())
			defer testDB.Close()

			t := TestFile{Path: bf, Contents: contents}
			Expect(t.Create()).To(Succeed())
			defer t.Remove()

			err = Clean([]string{}, &packages, cache, bf, Flags{DryRun: false}, testDB.DB)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(brew.ErrCouldNotFindPackageInfo("a2ps").Error()))
		})

		It("Should write out a new Brewfile in alphabetical order split into tap, brew, cask and mas sections", func() {
			expectedContents := `tap 'homebrew/bundle'
tap 'homebrew/core'

brew 'a2ps'

cask 'firefox'
cask 'google-chrome'

mas 'Xcode', id: 497799835`

			testDB, err := NewTestDB(dbFile)
			Expect(err).ToNot(HaveOccurred())
			defer testDB.Close()

			testDB.AddTestBrewsByName("a2ps")

			t := TestFile{Path: bf, Contents: contents}
			Expect(t.Create()).To(Succeed())
			defer t.Remove()

			Clean([]string{}, &packages, cache, bf, Flags{DryRun: false}, testDB.DB)

			bytes, error := ioutil.ReadFile(bf)
			Expect(error).To(BeNil())

			Expect(bytes).To(Equal([]byte(expectedContents)))
		})

		It("Should not modify the existing Brewfile if the --dry-run flag is set", func() {
			testDB, err := NewTestDB(dbFile)
			Expect(err).ToNot(HaveOccurred())
			defer testDB.Close()

			testDB.AddTestBrewsByName("a2ps")

			t := TestFile{Path: bf, Contents: contents}
			Expect(t.Create()).To(Succeed())
			defer t.Remove()

			_ = captureStdout(func() {
				Clean([]string{}, &packages, cache, bf, Flags{DryRun: true}, testDB.DB)
			})

			bytes, error := ioutil.ReadFile(bf)
			Expect(error).To(BeNil())

			Expect(bytes).To(Equal([]byte(contents)))
		})

		It("Should output the cleaned Brewfile contents to stdout", func() {
			testDB, err := NewTestDB(dbFile)
			Expect(err).ToNot(HaveOccurred())
			defer testDB.Close()

			testDB.AddTestBrewsByName("a2ps")

			t := TestFile{Path: bf, Contents: contents}
			Expect(t.Create()).To(Succeed())
			defer t.Remove()

			expectedOutput := `tap 'homebrew/bundle'
tap 'homebrew/core'

brew 'a2ps'

cask 'firefox'
cask 'google-chrome'

mas 'Xcode', id: 497799835
`

			output := captureStdout(func() {
				Clean([]string{}, &packages, cache, bf, Flags{DryRun: true}, testDB.DB)
			})

			Expect(output).To(Equal(expectedOutput))
		})
	})
})
