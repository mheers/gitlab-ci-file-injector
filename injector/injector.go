package injector

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"gopkg.in/yaml.v2"
)

var prettyPrint bool = true

func processExpression(expression string) string {
	var prettyPrintExp = `(... | (select(tag != "!!str"), select(tag == "!!str") | select(test("(?i)^(y|yes|n|no|on|off)$") | not))  ) style=""`
	if prettyPrint && expression == "" {
		return prettyPrintExp
	} else if prettyPrint {
		return fmt.Sprintf("%v | %v", expression, prettyPrintExp)
	}
	return expression
}

func modifyYaml(expression, path string) error {

	f := path
	writeInPlaceHandler := yqlib.NewWriteInPlaceHandler(f)
	out, err := writeInPlaceHandler.CreateTempFile()
	if err != nil {
		return err
	}
	// need to indirectly call the function so  that completedSuccessfully is
	// passed when we finish execution as opposed to now
	defer func() { writeInPlaceHandler.FinishWriteInPlace(true) }()

	outputFormat := "yaml"
	format, err := yqlib.OutputFormatFromString(outputFormat)
	if err != nil {
		return err
	}

	printer := yqlib.NewPrinter(out, format, false, false, 2, false)

	streamEvaluator := yqlib.NewStreamEvaluator()

	err = streamEvaluator.EvaluateFiles(processExpression(expression), []string{f}, printer, false)
	if err != nil {
		return err
	}
	return nil
}

type ServiceFile struct {
	Job      string     `yaml:"job"`
	Services []*Service `yaml:"services"`
}

type Service struct {
	Alias       string   `yaml:"alias"`
	Exec        string   `yaml:"exec"`
	Interpreter string   `yaml:"interpreter"`
	Files       []string `yaml:"files"`
}

// ReadServiceFiles reads the serviceFiles from the yaml file
func ReadServiceFiles(path string) ([]*ServiceFile, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	serviceFiles := []*ServiceFile{}
	err = yaml.Unmarshal(d, &serviceFiles)
	if err != nil {
		return nil, err
	}
	return serviceFiles, nil
}

func renderServiceFileEntrypoint(sf []*ServiceFile, job, serviceAlias string) (string, error) {
	s, err := getService(sf, job, serviceAlias)
	if err != nil {
		return "", err
	}

	return s.renderEntrypoint()
}

func getService(sf []*ServiceFile, job, serviceAlias string) (*Service, error) {
	for _, j := range sf {
		if j.Job == job {
			for _, s := range j.Services {
				if s.Alias == serviceAlias {
					return s, nil
				}
			}
		}
	}
	return nil, nil
}

func (s *Service) renderEntrypoint() (string, error) {
	contents, err := s.readFileContents()
	if err != nil {
		return "", err
	}
	lines := ""
	keys := []string{}
	for key := range contents {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, k := range keys {
		content := contents[k]
		dstFilePath := getDstFilePath(k)
		template, err := renderTemplate(dstFilePath, content)
		if err != nil {
			return "", err
		}
		lines += template
		lines += "\n"
	}

	lines += fmt.Sprintf("exec %s", s.Exec)

	return lines, nil
}

func (s *Service) readFileContents() (map[string]string, error) {
	results := make(map[string]string)
	for _, f := range s.Files {
		srcFilePath := getSrcFilePath(f)
		srcFileContentBytes, err := ioutil.ReadFile(srcFilePath)
		if err != nil {
			return nil, err
		}
		srcFileContent := string(srcFileContentBytes)
		err = checkFileContent(srcFileContent)
		if err != nil {
			return nil, err
		}
		results[f] = srcFileContent
	}
	return results, nil
}

func checkFileContent(content string) error {
	for i, line := range strings.Split(content, "\n") {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			return errors.New(fmt.Sprintf("file contains ending spaces or tabs: line %d", i+1))
		}
	}
	return nil
}

func getSrcFilePath(f string) string {
	return strings.Split(f, ":")[0]
}
func getDstFilePath(f string) string {
	return strings.Split(f, ":")[1]
}

func renderTemplate(path, content string) (string, error) {
	specTemplate := `
# inject {{ .Base }}
mkdir -p {{ .Folder }}
cat <<EOF > {{ .Path }}
{{ .Content }}
EOF
`

	tmpl, err := template.New("spec").
		Funcs(sprig.TxtFuncMap()).
		Parse(specTemplate)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl,
		struct {
			Content string
			Folder  string
			Path    string
			Base    string
		}{
			Content: content,
			Folder:  filepath.Dir(path),
			Path:    path,
			Base:    filepath.Base(path),
		})
	if err != nil {
		return "", err
	}
	return tpl.String(), nil

}

func InjectFilesIntoGitlabCIYaml(sf []*ServiceFile, path string) error {
	for _, j := range sf {
		for _, s := range j.Services {
			ep, err := s.renderEntrypoint()
			if err != nil {
				return err
			}
			err = writeEntypoint(j.Job, s, ep, path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func writeEntypoint(job string, service *Service, ep, path string) error {
	err := os.Setenv("EP", ep)
	if err != nil {
		return err
	}
	expression := fmt.Sprintf("(.%s.services.[] | select(.alias == \"%s\")).entrypoint |= [\"%s\", \"-c\", strenv(EP)]", job, service.Alias, service.Interpreter)
	fmt.Println(expression)
	err = modifyYaml(expression, path)
	if err != nil {
		return err
	}
	return nil
}
