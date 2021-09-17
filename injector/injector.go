package injector

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
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
var compressAndEncode bool = true

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
	Compressed  bool     `yaml:"compressed"`
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
	contentBytesMap, err := s.readFileContents()
	if err != nil {
		return "", err
	}

	keys := []string{}
	for key := range contentBytesMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var data string

	lines := ""
	for _, k := range keys {
		contentBytes := contentBytesMap[k]
		dstFilePath := getDstFilePath(k)

		if s.Compressed {
			c, err := compress(contentBytes)
			if err != nil {
				return "", err
			}
			data = encode(c)
		} else {
			data = string(contentBytes)
			err = checkFileContent(data)
			if err != nil {
				srcFilePath := getSrcFilePath(k)
				return "", fmt.Errorf("error in %s: %s", srcFilePath, err.Error())
			}
		}

		template, err := renderTemplate(dstFilePath, data, s.Compressed)
		if err != nil {
			return "", err
		}
		lines += template
		lines += "\n"
	}

	lines += fmt.Sprintf("exec %s", s.Exec)

	return lines, nil
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(data)); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Service) readFileContents() (map[string][]byte, error) {
	results := make(map[string][]byte)
	for _, f := range s.Files {
		srcFilePath := getSrcFilePath(f)
		srcFileContentBytes, err := ioutil.ReadFile(srcFilePath)
		if err != nil {
			return nil, err
		}
		results[f] = srcFileContentBytes
	}
	return results, nil
}

func checkFileContent(content string) error {
	for i, line := range strings.Split(content, "\n") {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			return fmt.Errorf("file contains ending spaces or tabs: line %d", i+1)
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

func renderTemplate(path, content string, compressed bool) (string, error) {
	specTemplate := `
# inject {{ .Base }}
mkdir -p {{ .Folder }}
cat <<EOF > {{ .Path }}
{{ .Content }}
EOF
`
	if compressed {
		specTemplate = `
# inject {{ .Base }}
mkdir -p {{ .Folder }}
echo {{ .Content }} | base64 -d | gunzip > {{ .Path }}
`
	}

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
