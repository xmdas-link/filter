package model

import (
	"bufio"
	"os"
	"strings"
)

type Model struct {
	Policy [][]string
}

type PolicyRule struct {
	Sub     string
	Model   string
	Fields  []string
	Encoder Encoder
}

func (m *Model) loadPolicy(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m.loadPolicyLine(line)
	}

	return scanner.Err()
}

func (m *Model) loadPolicyLine(line string) {
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	tokens := strings.Split(line, ",")
	for i := 0; i < len(tokens); i++ {
		tokens[i] = strings.TrimSpace(tokens[i])
	}

	m.Policy = append(m.Policy, tokens)
}

func NewModelFromFile(modelPath string) (*Model, error) {
	m := &Model{}

	if err := m.loadPolicy(modelPath); err != nil {
		return nil, err
	}

	return m, nil
}
