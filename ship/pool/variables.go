package pool

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"sync"
)

type Variables struct {
	lock     sync.RWMutex
	location string
	storage  []EnvVariable
}

type EnvVariable struct {
	Key   string    `yaml:"key"`
	Node  string    `yaml:"node"`
	Value yaml.Node `yaml:"val"`
}

type ShowVariable struct {
	Name string   `json:"name"`
	Node string   `json:"node"`
	Size int      `json:"size"`
	Uses []string `json:"uses"`
}

func NewVariables(dirManifests string) (*Variables, error) {
	location := dirManifests + "/_variables.yaml"
	if _, err := os.Stat(location); os.IsNotExist(err) {
		yamlData, err := yaml.Marshal(&[]EnvVariable{})
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(location, yamlData, 0600)
		if err != nil {
			return nil, err
		}
	}

	buf, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var vars []EnvVariable
	err = yaml.Unmarshal(buf, &vars)
	if err != nil {
		return nil, fmt.Errorf("variables unmarshal err: %v", err)
	}

	return &Variables{location: location, storage: vars}, nil
}

func (v *Variables) Get(key, node string) string {
	v.lock.RLock()
	defer v.lock.RUnlock()

	for _, s := range v.storage {
		if s.Key == key && s.Node == node {
			return s.Value.Value
		}
	}
	return ""
}

func (v *Variables) ListKeys() []ShowVariable {
	v.lock.RLock()
	defer v.lock.RUnlock()

	list := make([]ShowVariable, 0, len(v.storage))
	for _, s := range v.storage {
		list = append(list, ShowVariable{
			Name: s.Key,
			Node: s.Node,
			Size: len(s.Value.Value),
		})
	}
	return list
}

func (v *Variables) Set(key, node, value string) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	for _, s := range v.storage {
		if s.Key == key && s.Node == node {
			return errors.New("variable already exists")
		}
	}

	v.storage = append(v.storage, EnvVariable{
		Key:  key,
		Node: node,
		Value: yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.LiteralStyle,
			Value: strings.TrimSpace(value),
		},
	})

	return v.commit()
}

func (v *Variables) Delete(key, node string) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	evs := make([]EnvVariable, 0, len(v.storage))
	for _, s := range v.storage {
		if s.Key == key && s.Node == node {
			continue
		}
		evs = append(evs, s)
	}
	v.storage = evs

	return v.commit()
}

func (v *Variables) Edit(key, node, newName, newNode string) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	for _, s := range v.storage {
		if s.Key == newName && s.Node == newNode {
			return errors.New("variable already exists")
		}
	}

	for k, s := range v.storage {
		if s.Key == key && s.Node == node {
			v.storage[k].Node = newNode
			v.storage[k].Key = newName
			break
		}
	}

	return v.commit()
}

func (v *Variables) commit() error {
	data, err := yaml.Marshal(v.storage)
	if err != nil {
		return err
	}

	//data2 := strings.ReplaceAll(string(data), "  ", " ")

	err = os.WriteFile(v.location, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
