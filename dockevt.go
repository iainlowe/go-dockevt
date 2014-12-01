// Package dockevt provides a way to hook into the stream of events sent by Docker.

package dockevt

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// An event from the Docker daemon
type Event struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Container *Container
}

// The Docker container for which an event occurred
type Container struct {
	ID              string
	Name            string
	Image           string
	Config          *Config
	NetworkSettings *NetworkSettings
}

type Config struct {
	Hostname string
	Env      []string
}

type NetworkSettings struct {
	IPAddress   string
	PortMapping map[string]map[string]string
}

func (c Container) HostID() string {
	return strings.Split(c.Config.Hostname, ".")[0]
}

func (c Container) IP() string {
	return c.NetworkSettings.IPAddress
}

func Watch(events chan Event) {
	c := http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
	res, err := c.Get("http://localhost/events?since=1")
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	d := json.NewDecoder(res.Body)

	for {
		var evt Event
		if err := d.Decode(&evt); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln(err)
		}
		if ct := inspectContainer(evt.ID, c); ct != nil {
			evt.Container = ct
			events <- evt
		}
	}

	close(events)
}

func inspectContainer(id string, c http.Client) *Container {
	// Use the Container id to fetch the Container json from the Remote API
	// http://docs.docker.io/en/latest/api/docker_remote_api_v1.4/#inspect-a-Container
	res, err := c.Get("http://localhost/containers/" + id + "/json")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		d := json.NewDecoder(res.Body)

		var Container Container
		if err = d.Decode(&Container); err != nil {
			log.Println("error decoding docker event JSON:", err)
		}
		return &Container
	}
	return nil
}
