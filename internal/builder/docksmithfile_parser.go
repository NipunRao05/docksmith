package builder

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"

	"docksmith/internal/model"
)

var validInstructions = map[string]bool{
	"FROM":    true,
	"COPY":    true,
	"RUN":     true,
	"WORKDIR": true,
	"ENV":     true,
	"CMD":     true,
}

func ParseDocksmithfile(path string) ([]model.Instruction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	var instructions []model.Instruction

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		instType := strings.ToUpper(parts[0])

		if !validInstructions[instType] {
			return nil, errors.New("invalid instruction at line " + strconv.Itoa(lineNum) + ": " + instType)
		}

		instruction := model.Instruction{
			Type: instType,
			Args: parts[1:],
			Raw:  line,
			Line: lineNum,
		}

		instructions = append(instructions, instruction)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return instructions, nil
}
