package injector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadServiceFiles(t *testing.T) {
	sf, err := ReadServiceFiles("./test.yml")
	assert.Nil(t, err)
	assert.NotNil(t, sf)
}

func TestGetService(t *testing.T) {
	s, err := getService(getDemoServiceFiles(), "demo-job", "service-job")
	assert.Nil(t, err)
	assert.NotNil(t, s)
}

func TestRenderServiceFileEntrypoint(t *testing.T) {
	template, err := renderServiceFileEntrypoint(getDemoServiceFiles(), "demo-job", "service-job")
	assert.Nil(t, err)
	assert.NotEmpty(t, template)
}

func TestRenderTemplate(t *testing.T) {
	template, err := renderTemplate("/var/log/syslog", "juhu, it is a log line")
	assert.Nil(t, err)
	assert.NotEmpty(t, template)
}

func getDemoServiceFiles() []*ServiceFile {
	return []*ServiceFile{
		{
			Job: "demo-job",
			Services: []*Service{
				{
					Alias: "service-job",
					Files: []string{
						"/tmp/test.txt:/dst/ok.txt",
					},
				},
			},
		},
	}
}

func TestSprintf(t *testing.T) {
	s := `
		# inject {{ .Base }}
		mkdir -p {{ .Folder }}
		cat <<EOF > {{ .Path }}
		{{ .Content }}
		EOF
		`
	f := fmt.Sprintf("%s ok", s)
	t.Log(f)
}

func TestCheckFileContent(t *testing.T) {
	content := `
ok
here 
is sth broken
`
	err := checkFileContent(content)
	assert.NotNil(t, err)

	content2 := `
ok
here
is sth broken
`
	err = checkFileContent(content2)
	assert.Nil(t, err)
}
