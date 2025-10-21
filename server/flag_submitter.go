package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + fmt.Sprintf("... (截断, 共计 %d 字节)", len(output))
}

func submitFlagToCompetition(flag string, clientIP string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("提交flag时出现问题: %v", r)
			}
		}()

		template := getConfigValue("submit_template")
		if template == "" {
			log.Printf("来自 %s 的Flag: %s, 错误: 提交模板未配置", clientIP, flag)
			addMessage("warning", "Flag提交失败", fmt.Sprintf("来自 %s 的Flag: %s, 错误: 提交模板未配置", clientIP, flag))
			return
		}

		command := strings.ReplaceAll(template, "{FLAG}", flag)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else if ctx.Err() == context.DeadlineExceeded {
				log.Printf("来自 %s 的Flag: %s, 错误: 命令执行超时", clientIP, flag)
				addMessage("warning", "Flag提交失败", fmt.Sprintf("来自 %s 的Flag: %s, 错误: 命令执行超时", clientIP, flag))
				return
			} else {
				exitCode = -1
			}
		}

		stdoutStr := truncateOutput(stdout.String(), 500)
		stderrStr := truncateOutput(stderr.String(), 500)

		if err != nil {
			log.Printf("来自 %s 的Flag: %s, 命令执行失败 [退出码: %d], stderr: %s",
				clientIP, flag, exitCode, stderrStr)
			addMessage("warning", "Flag提交失败",
				fmt.Sprintf("来自 %s 的Flag: %s, 错误: 退出码 %d", clientIP, flag, exitCode))
			return
		}

		log.Printf("成功提交来自 %s 的Flag: %s [退出码: %d]", clientIP, flag, exitCode)

		if stdout.Len() > 0 {
			log.Printf("Flag提交命令输出: %s", stdoutStr)
		}
		if stderr.Len() > 0 {
			log.Printf("Flag提交命令stderr: %s", stderrStr)
		}

		addMessage("success", "Flag已提交", fmt.Sprintf("成功提交来自 %s 的Flag: %s", clientIP, flag))
	}()
}
