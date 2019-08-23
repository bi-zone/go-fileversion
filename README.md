# go-fileversion


Package `fileversion` provides wrapper for windows version-information resource.

Using the package you can extract the following info:

![](https://github.com/bi-zone/go-fileversion/blob/version_info_fillinf/assets/explorer_properties.png)



## Examples

Print version info from input file
```golang
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
	fmt.Println("FileVeLegalTrademarksrsion:", f.FileVeLegalTrademarksrsion())
	fmt.Println("PrivateBuild:", f.PrivateBuild())
	fmt.Println("SpecialBuild:", f.SpecialBuild())

	fmt.Printf("\n%+#v\n", f.GetFixedInfo())

	fmt.Printf("%+#v\n", f.Locales)
}
```

You can choose the locale for getting property.

Choose locale from object:

```golang
f, err := fileversion.New(os.Args[1])
if err != nil {
    log.Fatal(err)
}
if len(f.Locales) > 1 {
    fmt.Println(f.GetPropertyWithLocale("PropertyName", f.Locales[len(f.Locales) - 1]))
}
```

Also you can get property with owen-defined locale:
```golang
f, err := fileversion.New(os.Args[1])
if err != nil {
    log.Fatal(err)
}
germanLocale := fileversion.Locale{
		LangID: 0x0407,
		CharsetID: fileversion.CSUnicode,
	}
fmt.Println(f.GetPropertyWithLocale("PropertyName", germanLocale))
```
But we don't recommend to do this :) If object doesn't have locale, 
you will get an error.

 `f.GetProperty` method tries to get property with different locales given 
 windows features.
The idea of locales handling was copied from 
[.NET Framework 4.8](https://referencesource.microsoft.com/#System/services/monitoring/system/diagnosticts/FileVersionInfo.cs,036c54a4aa10d39f,references)

```golang
f, err := fileversion.New(os.Args[1])
if err != nil {
    log.Fatal(err)
}
f.GetProperty("PropertyName")
```
