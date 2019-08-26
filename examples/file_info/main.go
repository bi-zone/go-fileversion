package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bi-zone/go-fileversion"
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
	fmt.Println("LegalTrademarks:", f.LegalTrademarks())
	fmt.Println("PrivateBuild:", f.PrivateBuild())
	fmt.Println("SpecialBuild:", f.SpecialBuild())

	fixedInfo := f.FixedInfo()
	fmt.Printf("\n%+#v\n", fixedInfo)
	fmt.Println("File version:", fixedInfo.FileVersion)
	fmt.Println("Product version:", fixedInfo.ProductVersion)

	fmt.Printf("%+#v\n", f.Locales)

	germanLocale := fileversion.Locale{
		LangID:    0x0407,
		CharsetID: fileversion.CSUnicode,
	}
	fmt.Println(f.GetPropertyWithLocale("PropertyName", germanLocale))
}
