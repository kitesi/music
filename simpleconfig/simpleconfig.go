// helper for simple config files (key=value)
package simpleconfig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	values      map[string]string
	fileName    string
	allowedKeys map[string]struct{}
}

func NewConfig(fileName string, allowedKeys []string) (Config, error) {
	set := make(map[string]struct{})
	for _, key := range allowedKeys {
		set[key] = struct{}{}
	}

	cfg := Config{
		values:      make(map[string]string),
		fileName:    fileName,
		allowedKeys: set,
	}

	_, err := os.Stat(fileName)

	if os.IsNotExist(err) {
		_, err = os.Create(fileName)

		if err != nil {
			return cfg, errors.New("Error with creating config file: " + err.Error())
		}

		fmt.Println("Created credentials file at " + fileName)
	} else if err != nil {
		return cfg, errors.New("Error with stating config file: " + err.Error())
	}

	err = cfg.ReadConfig()
	return cfg, err
}

func (c *Config) ReadConfig() error {
	file, err := os.Open(c.fileName)

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "=")

		if len(parts) == 2 {
			key := parts[0]

			if c.allowedKeys != nil {
				if _, ok := c.allowedKeys[key]; !ok {
					return fmt.Errorf("Invalid key: %s in %s", key, c.fileName)
				}
			}

			c.values[key] = parts[1]
		} else {
			return fmt.Errorf("Invalid line: %s in %s", line, c.fileName)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (c *Config) WriteConfig() error {
	file, err := os.OpenFile(c.fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		return err
	}

	defer file.Close()
	writer := bufio.NewWriter(file)

	for key, value := range c.values {
		line := fmt.Sprintf("%s=%s\n", key, value)
		_, err := writer.WriteString(line)

		if err != nil {
			return err
		}
	}

	writer.Flush()

	return nil
}

func (c *Config) Get(key string) (string, bool) {
	val, ok := c.values[key]
	return val, ok
}

func (c *Config) Set(key, value string) error {
	if c.allowedKeys != nil {
		if _, ok := c.allowedKeys[key]; !ok {
			return fmt.Errorf("Invalid key: %s in %s", key, c.fileName)
		}
	}

	c.values[key] = value
	return nil
}
