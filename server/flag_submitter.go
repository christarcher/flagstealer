package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

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

		time.Sleep(90 * time.Second)

		// 替换模板中的{FLAG}占位符
		command := strings.ReplaceAll(template, "{FLAG}", flag)

		// 使用sh执行命令
		cmd := exec.Command("sh", "-c", command)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// 执行命令
		err := cmd.Run()

		if err != nil {
			log.Printf("来自 %s 的Flag: %s, 错误: 命令执行失败: %v, stderr: %s", clientIP, flag, err, stderr.String())
			addMessage("warning", "Flag提交失败", fmt.Sprintf("来自 %s 的Flag: %s, 错误: %v", clientIP, flag, err))
			return
		}

		// 记录命令输出
		if stdout.Len() > 0 {
			log.Printf("Flag提交命令输出: %s", stdout.String())
		}
		if stderr.Len() > 0 {
			log.Printf("Flag提交命令stderr: %s", stderr.String())
		}

		log.Printf("成功提交来自 %s 的Flag: %s", clientIP, flag)
		addMessage("success", "Flag已提交", fmt.Sprintf("成功提交来自 %s 的Flag: %s", clientIP, flag))
	}()
}
