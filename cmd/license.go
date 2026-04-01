package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage automatically loadable licenses",
}

var licenseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the licenses that will be auto-loaded",

	RunE: func(cmd *cobra.Command, args []string) error {
		licenses, err := findLicenses()
		if err != nil {
			return err
		}

		for _, l := range licenses {
			fmt.Printf("%s will be loaded as default value for environment variable %s\n", l, licenseEnvVarKey(l))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(licenseCmd)
	licenseCmd.AddCommand(licenseListCmd)
}

func findLicenses() ([]string, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return []string{}, err
	}

	return filepath.Glob(filepath.Join(homePath, ".shikari", "*.hclic"))
}

func licenseEnvVarKey(path string) string {
	licenseFilename := filepath.Base(path)
	licenseName := strings.TrimSuffix(licenseFilename, filepath.Ext(licenseFilename))
	return strings.ToUpper(licenseName) + "_LICENSE"
}
