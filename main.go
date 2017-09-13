package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"sort"
	"text/template"
	"time"

	"github.com/Nitro/sidecar/service"
	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/relistan/rubberneck.v1"
)

const LoopDelayInterval = 3 * time.Second

type Config struct {
	RefreshInterval time.Duration `envconfig:"REFRESH_INTERVAL" default:"5s"`
	FollowService   string        `envconfig:"FOLLOW_SERVICE" default:"lazyraster"`
	FollowPort      int64         `envconfig:"FOLLOW_PORT" required:"true"`
	TemplateFile    string        `envconfig:"TEMPLATE_FILENAME" default:"templates/nginx.conf.tmpl"`
	UpdateCommand   string        `envconfig:"UPDATE_COMMAND"`
	ValidateCommand string        `envconfig:"VALIDATE_COMMAND"`
	SidecarAddress  string        `envconfig:"SIDECAR_ADDRESS" required:"true"`
	NginxConf       string        `envconfig:"NGINX_CONF" default:"/nginx/nginx.conf"`
}

type ApiServices struct {
	Services map[string][]*service.Service
}

func WriteTemplate(config *Config, servers []string, output io.Writer) error {
	funcMap := template.FuncMap{
		"now":     time.Now().UTC,
		"servers": func() []string { return servers },
	}

	t, err := template.New("haproxy").Funcs(funcMap).ParseFiles(config.TemplateFile)
	if err != nil {
		return fmt.Errorf("Error parsing template '%s': %s", config.TemplateFile, err)
	}

	err = t.ExecuteTemplate(output, path.Base(config.TemplateFile), nil)
	if err != nil {
		return fmt.Errorf("Error executing template '%s': %s", config.TemplateFile, err)
	}

	return nil
}

// run executes a command and bubbles up the error.
func run(command string) error {
	cmd := exec.Command("/bin/bash", "-c", command)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	if err != nil {
		err = fmt.Errorf("Error running '%s': %s\n%s\n%s", command, err, stdout, stderr)
	}

	return err
}

func innerUpdate(config *Config, previousServers []string) ([]string, error) {
	servers, err := FetchServers(config)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch updated server list! (%s)", err)
	}

	if reflect.DeepEqual(servers, previousServers) {
		return servers, nil
	}

	output, err := os.OpenFile(config.NginxConf, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("Unable to open output file for writing: %s", err)
	}

	err = WriteTemplate(config, servers, output)
	if err != nil {
		return nil, fmt.Errorf("Unable to write template: %s", err)
	}

	log.Info("Reloading Nginx config...")

	previousServers = servers

	err = run(config.ValidateCommand)
	if err != nil {
		return nil, fmt.Errorf("Unable to validate nginx config! (%s)", err)
	}

	err = run(config.UpdateCommand)
	if err != nil {
		return nil, fmt.Errorf("Unable to reload nginx config! (%s)", err)
	}

	return servers, nil
}

func UpdateNginx(config *Config) {
	var previousServers []string
	var err error

	for {
		previousServers, err = innerUpdate(config, previousServers)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(LoopDelayInterval)
	}
}

func findPortWithSvcPortNumber(ports []service.Port, config *Config) string {
	for _, port := range ports {
		// Short circuit on the first port that matches
		if port.ServicePort == config.FollowPort {
			return fmt.Sprintf("%s:%d", port.IP, port.Port)
		}
	}

	return ""
}

// FetchServers will connect to Sidecar, and with a timeout, fetch and
// parse the resulting structure. It will return a list of only the
// server:port combinations for the queried service
func FetchServers(config *Config) ([]string, error) {
	client := &http.Client{Timeout: config.RefreshInterval * 2}
	url := "http://" + config.SidecarAddress + "/services/" + config.FollowService + ".json"

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiServices ApiServices
	err = json.Unmarshal(bytes, &apiServices)
	if err != nil {
		return nil, err
	}

	// We won't get here if there were no services, the Unmarshal should
	// fail instead because we get an ApiError instead.
	svcs := apiServices.Services[config.FollowService]

	var servers []string
	for _, svc := range svcs {
		portStr := findPortWithSvcPortNumber(svc.Ports, config)
		if len(portStr) < 1 {
			log.Warnf("Got no port match for service on hostname: %s",
				svc.Hostname,
			)
			continue
		}

		if svc.Status != 0 {
			log.Debugf("Skipping service with status %d on hostname: %s",
				svc.Status,
				svc.Hostname,
			)
			continue
		}
		servers = append(servers, portStr)
	}

	// These need to be sorted for later comparison
	sort.Strings(servers)

	return servers, nil
}

func main() {
	var config Config
	err := envconfig.Process("discovery", &config)
	if err != nil {
		log.Fatal(err)
	}

	// Set some defaults that are unpleasant to put in the struct definition
	if len(config.UpdateCommand) < 1 {
		config.UpdateCommand = `/nginx/nginx -c ` + config.NginxConf + ` -g "error_log /dev/fd/1;"`
	}

	if len(config.ValidateCommand) < 1 {
		config.UpdateCommand = "/bin/kill -HUP `cat /tmp/nginx.pid`"
	}

	rubberneck.Print(config)
	UpdateNginx(&config)
}
