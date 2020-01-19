package docradle

import (
	"fmt"
	"github.com/gookit/color"
	"go.pyspa.org/brbundle"
	"html/template"
	"io"
	"os"
	user2 "os/user"
)

var helpMessage = `
<green>Config file "docradle.%s" is generated successfully.</>

Run with the following command:

<gray>$</> <lightBlue>docradle run</> <yellow>your-command</> <yellow>options...</>
`

type templatValue struct {
	UserName string
}

func Generate(stdout io.Writer, format string) error {
	switch format {
	case "json":
		// do nothing
	case "cue":
		fallthrough
	case "yaml":
		color.Fprintf(os.Stderr, "<red>Init in %s format is not implemented yet</>\n", format)
		return fmt.Errorf("%s is not implemented yet", format)
	}

	f, err := os.Create(fmt.Sprintf("docradle.%s", format))
	if err != nil {
		return err
	}
	defer f.Close()

	e, err := brbundle.Find("sample.json")
	if err != nil {
		panic(err)
	}
	content, err := e.ReadAll()
	if err != nil {
		panic(err)
	}
	t := template.Must(template.New("sample").Parse(string(content)))
	user, err := user2.Current()
	err = t.Execute(f, &templatValue{UserName: user.Username})
	if err != nil {
		return err
	}
	color.Fprintf(stdout, helpMessage, format)
	return nil
}
