package helpers

import (
	"fmt"
	"runtime"

	"github.com/common-nighthawk/go-figure"
	"github.com/mheers/gitlab-ci-file-injector/config"
	"github.com/morikuni/aec"
)

// PrintInfo print Info
func PrintInfo() {
	f := figure.NewColorFigure("Gitlab CI File Injector", "big", "red", true)
	figletStr := f.String()
	PrintFiglet(figletStr)
	fmt.Println()

	fmt.Println("Contact:")
	fmt.Println(config.ContactStr)
	fmt.Println()
}

// PrintFiglet prints Figlet
func PrintFiglet(figletStr string) {
	figletColoured := aec.BlueF.Apply(figletStr)
	if runtime.GOOS == "windows" {
		figletColoured = aec.GreenF.Apply(figletStr)
	}
	fmt.Println(figletColoured)
}
