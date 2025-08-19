package cmd

import (
	"fmt"
)

func DisplayBanner() {
	fmt.Println("ooooooo  ooooo oooooooooooo  o8o                      .o88o. oooo              ")
	fmt.Println(" `8888    d8'  `888'     `8  `\"'                      888 `\" `888              ")
	fmt.Println("   Y888..8P     888         oooo  oooo d8b  .ooooo.  o888oo   888  oooo    ooo ")
	fmt.Println("    `8888'      888oooo8    `888  `888\"\"8P d88' `88b  888     888   `88.  .8'  ")
	fmt.Println("   .8PY888.     888    \"     888   888     888ooo888  888     888    `88..8'   ")
	fmt.Println("  d8'  `888b    888          888   888     888    .o  888     888     `888'    ")
	fmt.Println("o888o  o88888o o888o        o888o d888b    `Y8bod8P' o888o   o888o     .8'     ")
	fmt.Println("                                                                   .o..P'      ")
	//fmt.Println("                                                                   `Y8P'       ")
	fmt.Printf("    Version:%s  Author:%s  BuildDate:%s             `Y8P'\n\n", defaultVersion, defaultAuthor, defaultBuildDate)
}
