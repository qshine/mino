package mino

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func promptConfigValue(ctx context.Context, input *os.File, reader *bufio.Reader, output io.Writer, label, fallback string, secret bool) (value string, err error) {
	state, err := stty(input, "-g")
	if err != nil {
		return "", fmt.Errorf("Config is incomplete. Start Mino in an interactive terminal to finish setup before using piped input.")
	}
	if secret {
		// 恢复操作不用已取消的 context，确保 Ctrl+C 后终端仍能正常回显。
		defer func() {
			_, restoreErr := stty(input, state)
			fmt.Fprintln(output)
			if restoreErr != nil {
				err = fmt.Errorf("Failed to restore terminal echo. Run stty sane: %w", restoreErr)
			}
		}()
		if _, err := stty(input, "-echo"); err != nil {
			return "", fmt.Errorf("Failed to hide API key input: %w", err)
		}
	}
	for {
		prompt := label
		if fallback != "" {
			prompt += " [" + fallback + "]"
		}
		if secret {
			prompt += " (input hidden)"
		}
		if _, err := fmt.Fprint(output, prompt+": "); err != nil {
			return "", err
		}
		line, err := readConfigLine(ctx, reader)
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = fallback
		}
		if value == "" {
			fmt.Fprintln(output, "\n"+label+" is required. Please enter a value.")
			continue
		}
		if label == "API URL" {
			if err := validateBaseURL(value); err != nil {
				fmt.Fprintln(output, terminalText(err.Error()))
				continue
			}
		}
		return value, nil
	}
}

// 使用 macOS 自带 stty，仅传固定选项或它返回的状态，不执行用户输入。
func stty(input *os.File, args ...string) (string, error) {
	command := exec.Command("/bin/stty", args...)
	command.Stdin = input
	data, err := command.Output()
	return strings.TrimSpace(string(data)), err
}

func readConfigLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	line := make(chan inputLine, 1)
	go func() {
		text, err := reader.ReadString('\n')
		line <- inputLine{text: text, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-line:
		return result.text, result.err
	}
}
