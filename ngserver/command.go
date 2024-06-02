package ngserver

import (
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func getNpmRoot() string {
	cmd := exec.Command("npm", "root")
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Replace(out.String(), "\n", "", -1)
}

func getGlobalNpmRoot() string {
	cmd := exec.Command("npm", "root", "-g")
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Replace(out.String(), "\n", "", -1)
}

func angularlsCmd() (string, []string) {
	rootDir, _ := filepath.Abs(".")
	locations := []string{getGlobalNpmRoot(), getNpmRoot(), filepath.Join(rootDir, "node_modules")}

	args := []string{"--stdio", "--tsProbeLocations"}
	args = append(args, locations...)
	args = append(args, "--ngProbeLocations")
	args = append(args, locations...)

	return "ngserver", args
}
