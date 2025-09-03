package cli

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
	fmt.Printf("    Version:%s  Author:%s  BuildDate:%s            `Y8P'\n\n", defaultVersion, defaultAuthor, defaultBuildDate)
}

// 填充好的banner字符串
//var Banner = "ooooooo  ooooo oooooooooooo  o8o                      .o88o. oooo              \n" +
//	" `8888    d8'  `888'     `8  `\"'                      888 `\" `888              \n" +
//	"   Y888..8P     888         oooo  oooo d8b  .ooooo.  o888oo   888  oooo    ooo \n" +
//	"    `8888'      888oooo8    `888  `888\"\"8P d88' `88b  888     888   `88.  .8'  \n" +
//	"   .8PY888.     888    \"     888   888     888ooo888  888     888    `88..8'   \n" +
//	"  d8'  `888b    888          888   888     888    .o  888     888     `888'    \n" +
//	"o888o  o88888o o888o        o888o d888b    `Y8bod8P' o888o   o888o     .8'     \n" +
//	"                                                                   .o..P'      \n" +
//	"    Version:" + defaultVersion + "  Author:" + defaultAuthor + "  BuildDate:" + defaultBuildDate + "            `Y8P'\n\n"

var Banner = "   _  __ _______           ______     \n" +
	"  | |/ // ____(_)_______  / __/ /_  __\n" +
	"  |   // /_  / / ___/ _ \\/ /_/ / / / /\n" +
	" /   |/ __/ / / /  /  __/ __/ / /_/ / \n" +
	"/_/|_/_/   /_/_/   \\___/_/ /_/\\__, /  \n" +
	"                             /____/   \n\n"
