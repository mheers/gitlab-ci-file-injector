> Gitlab CI file injector

Inject files into gitlab ci services

# Installation

```bash
go install github.com/mheers/gitlab-ci-file-injector@latest
```

# Usage

```bash
glabci-fi -i serviceFile.yml -o .gitlab-ci.yml
```

# Use case

In a gitlab ci pipeline we can define services but these services can not be configured by mounting or copying (configuration)files into. This tool renders the contents of wanted files in to services entrypoint.

# Example

`.gitlab-ci.yml`:

```yaml
services:
  - docker:20.10.8-dind
stages:
  - test
test:
  stage: test
  image:
    name: alpine
  services:
    - name: httpd:2.4.48-alpine
      alias: proxy
  script:
    - sleep 240
```

`serviceFile.yml`:

```yaml
- job: test
  services:
    - alias: proxy
      exec: httpd-foreground
      interpreter: "/bin/sh"
      files:
        - ./examples/apache2/cert.pem:/usr/local/apache2/conf/server.crt
        - ./examples/apache2/key.pem:/usr/local/apache2/conf/server.key
        - ./examples/apache2/httpd-ssp.conf:/usr/local/apache2/conf/extra/httpd-ssp.conf
        - ./examples/apache2/httpd.conf:/usr/local/apache2/conf/extra/httpd.conf
```

After running
`gitlab-ci-file-injector inject -i examples/serviceFile.yml -o examples/.gitlab-ci.yml`
the `.gitlab-ci.yml` will almost look like this:

```yaml
services:
  - docker:20.10.8-dind
stages:
  - test
test:
  stage: test
  image:
    name: alpine
  services:
    - name: httpd:2.4.48-alpine
      alias: proxy
      entrypoint:
        - /bin/sh
        - -c
        - |2-
          # inject key.pem
          mkdir -p examples/apache2
          cat <<EOF > ./examples/apache2/key.pem
          -----BEGIN PRIVATE KEY-----
          ...some secrets....
          -----END PRIVATE KEY-----

          EOF

          exec httpd-foreground
  script:
    - sleep 240
```

with all the file contents rendered to the proxy services entrypoint.

# TODO:

- [ ] correct error logging
- [x] print file path where lines with ending spaces are
- [ ] restructure and clean up code
