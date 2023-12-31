package data

import (
	"errors"
	"fmt"
	"github.com/samber/lo"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	BaseConfig
	Hosts []Host `yaml:"hosts"`
}
type BaseConfig struct {
	Http   int    `yaml:"http"`
	Https  int    `yaml:"https"`
	API    string `yaml:"api"`
	Secret string `yaml:"secret"`
}
type Host struct {
	Host     string    `yaml:"host"`
	Forwards []Forward `yaml:"forwards"`
}
type Forward struct {
	Type             string `yaml:"type"`
	ForwardWeb       `yaml:",inline"`
	ForwardPortRange `yaml:"port_range,omitempty"`
	ForwardPort      `yaml:",inline"`
}

func (o *Forward) Validate() error {
	switch o.Type {
	case ForwardTypeWeb:
		return o.ForwardWeb.Validate()
	case ForwardTypePort:
		return o.ForwardPort.Validate()
	case ForwardTypePortRange:
		_, err := o.ForwardPortRange.GetPorts()
		return err
	default:
		return errors.New("type is not defined: " + o.Type)
	}
}

type ForwardPortRange string

func (f *ForwardPortRange) GetPorts() ([]int, error) {
	arr := strings.Split(strings.ReplaceAll(string(*f), " ", ""), ",")
	var ports []int
	for _, seg := range arr {
		arr2 := strings.Split(seg, "-")
		if len(arr2) == 1 {
			p, err := strconv.Atoi(seg)
			if err != nil {
				return nil, err
			}
			ports = append(ports, p)
		} else if len(arr2) == 2 {
			start, err := strconv.Atoi(arr2[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(arr2[1])
			if err != nil {
				return nil, err
			}
			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			return nil, errors.New("malformed: " + seg)
		}
	}
	return ports, nil
}

type ForwardWeb struct {
	Http      int      `yaml:"http"`
	Https     int      `yaml:"https"`
	Hostnames []string `yaml:"hostnames"`
}

func (o *ForwardWeb) Validate() error {
	if o.Http == 0 {
		o.Http = 80
	}
	if o.Https == 0 {
		o.Https = 443
	}
	return lo.Ternary(len(o.Hostnames) == 0, errors.New("hostnames being empty"), nil)
}

type ForwardPort struct {
	Src int `yaml:"src"`
	Dst int `yaml:"dst"`
}

func (o *ForwardPort) Validate() error {
	if o.Src == 0 || o.Dst == 0 {
		return errors.New("src or dst not set")
	}
	return nil
}

func (o *Config) Validate() error {
	if o.Http == 0 {
		o.Http = 80
	}
	if o.Https == 0 {
		o.Https = 443
	}
	if o.API == "" {
		o.API = "127.0.0.1:2035"
	}
	for i := range o.Hosts {
		host := &o.Hosts[i]
		if _, err := net.LookupHost(host.Host); err != nil {
			return fmt.Errorf("error parsing host %s: %w", host.Host, err)
		}
		for j := range host.Forwards {
			forward := &host.Forwards[j]
			if err := forward.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}
