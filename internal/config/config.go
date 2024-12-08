package config

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
)

type Config struct {
	TorPath         string `json:"TorPath"`
	LyrebirdPath    string `json:"LyrebirdPath"`
	DubbingSavePath string `json:"DubbingSavePath"`
	Bridge          string `json:"Bridge"`
}

const configPath = "config.json"

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Save() error {
	file, err := os.Create(configPath)
	if err != nil {
		return errors.New("Can't create config file: " + err.Error())
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return errors.New("Can't save config file: " + err.Error())
	}

	return nil
}

func (c *Config) Load() error {
	file, err := os.Open(configPath)
	if err != nil {
		return errors.New("Can't open config file: " + err.Error())
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(c); err != nil {
		return errors.New("Can't load config file: " + err.Error())
	}

	return nil
}

func (c *Config) SetField(fieldName string, value any) error {
	v := reflect.ValueOf(c).Elem()
	field := v.FieldByName(fieldName)

	if !field.IsValid() {
		return errors.New("Can't find field " + fieldName)
	}

	if !field.CanSet() {
		return errors.New("Can't set field " + fieldName)
	}

	val := reflect.ValueOf(value)
	if field.Type() != val.Type() {
		return errors.New("Wrong type for field " + fieldName)
	}
	field.Set(val)

	return c.Save()
}
