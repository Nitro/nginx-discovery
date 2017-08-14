package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

const (
	ValidResponse = `
		{
		    "Services": {
		        "foo": [
		            {
		                "ID": "afc06fd44f8f",
		                "Name": "foo",
		                "Image": "gonitro/foo",
		                "Created": "2017-08-04T18:50:09Z",
		                "Hostname": "bede",
		                "Ports": [
		                    {
		                        "Type": "tcp",
		                        "Port": 26858,
		                        "ServicePort": 10101,
		                        "IP": "10.10.10.10"
		                    }
		                ],
		                "Updated": "2017-08-14T15:30:23.451172154Z",
		                "ProxyMode": "http",
		                "Status": 0
		            }
		        ]
		    },
		    "ClusterName": "foobar"
		}`
)

func Test_WriteTemplate(t *testing.T) {
	Convey("WriteTemplate()", t, func() {
		config := Config{
			TemplateFile: "templates/nginx.conf.tmpl",
		}

		servers := []string{
			"bocaccio:12345",
			"bede:3456",
		}

		Convey("Can write a file with the right servers", func() {
			buf := bytes.NewBuffer(make([]byte, 65535))
			err := WriteTemplate(&config, servers, buf)

			So(err, ShouldBeNil)
			So(buf.String(), ShouldContainSubstring, "server bocaccio:12345")
			So(buf.String(), ShouldContainSubstring, "server bede:3456")
		})

		Convey("Raises an error when the template is bad", func() {
			buf := bytes.NewBuffer(make([]byte, 65535))
			config.TemplateFile = "OMGTheresNoWayThisExists.OMG.OMG"
			err := WriteTemplate(&config, servers, buf)

			So(err, ShouldNotBeNil)
		})
	})
}

func Test_UpdateNginx(t *testing.T) {
	Convey("UpdateNginx()", t, func() {
		previousServers := []string{}

		outFile, _ := ioutil.TempFile("", "UpdateNginx")

		config := Config{
			TemplateFile:   "templates/nginx.conf.tmpl",
			SidecarAddress: "beowulf:31337",
			FollowService:  "foo",
			NginxConf:      outFile.Name(),
		}

		servers := []string{
			"10.10.10.10:26858",
		}

		httpmock.RegisterResponder("GET", "http://beowulf:31337/services/foo.json",
			httpmock.NewStringResponder(
				200, ValidResponse,
			),
		)

		httpmock.Activate()

		Reset(func() {
			httpmock.Reset()
			os.Remove(outFile.Name())
		})

		Convey("Writes a template when the servers changed", func() {
			newServers, err := innerUpdate(&config, previousServers)

			stat, _ := outFile.Stat()
			So(err, ShouldBeNil)
			So(stat.Size(), ShouldBeGreaterThan, 0)
			So(newServers, ShouldResemble, servers)
		})

		Convey("Does not write a template when the servers are the same", func() {
			newServers, err := innerUpdate(&config, servers)

			stat, _ := outFile.Stat()
			So(err, ShouldBeNil)
			So(stat.Size(), ShouldEqual, 0)
			So(newServers, ShouldResemble, servers)
		})

		Convey("Bubbles up errors in validation", func() {
			config.ValidateCommand = "false"

			_, err := innerUpdate(&config, previousServers)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Unable to validate")
		})

		Convey("Bubbles up errors when restarting", func() {
			config.ValidateCommand = "true"
			config.UpdateCommand = "false"

			_, err := innerUpdate(&config, previousServers)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Unable to reload")
		})
	})
}
