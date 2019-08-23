package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bi-zone/go-fileversion"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./file_info.exe <image-path>")
	}
	f, err := fileversion.New(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("CompanyName:", f.CompanyName())
	fmt.Println("FileDescription:", f.FileDescription())
	fmt.Println("FileVersion:", f.FileVersion())
	fmt.Println("InternalName:", f.InternalName())
	fmt.Println("LegalCopyright:", f.LegalCopyright())
	fmt.Println("OriginalFilename:", f.OriginalFilename())
	fmt.Println("ProductName:", f.ProductName())
	fmt.Println("ProductVersion:", f.ProductVersion())
	fmt.Println("Comments:", f.Comments())
	fmt.Println("FileVeLegalTrademarksrsion:", f.FileVeLegalTrademarksrsion())
	fmt.Println("PrivateBuild:", f.PrivateBuild())
	fmt.Println("SpecialBuild:", f.SpecialBuild())

	spew.Dump(f.FixedFileInfo)
}
